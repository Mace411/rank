package dao

import (
	"../base"
	"../config"
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"gopkg.in/mgo.v2/bson"
	"log"
	"strconv"
	"time"
)

var ctx = context.Background()

/*
添加送礼记录
*/
func AddSendGiftLog(gift base.Gift) (msg string, err error) {
	gift.Timestamp = time.Now()
	//先写db再写缓存
	session := connectMongo()
	defer session.Close()
	defer Reduce()
	db := session.DB(dbName)
	collection := db.C(giftCollection)
	err = collection.Insert(&gift)
	if err != nil {
		log.Println(base.SendGiftFailed, err)
		return base.SendGiftFailed, err
	}
	redisConn := connectRedis()
	defer redisConn.Close()
	redisConn.Del(ctx, strconv.Itoa(gift.Receiver))
	return base.Success, err
}

/*
更新排行榜
*/
func UpdateRank(gift base.Gift) (msg string, err error) {
	conn := connectRedis()
	defer conn.Close()
	value := config.GiftConfigMap[gift.GiftType]
	if value <= 0 {
		log.Println(base.GiftTypeNotExist)
		return base.GiftTypeNotExist, err
	}
	err = conn.ZIncrBy(ctx, base.RANK, float64(value), strconv.Itoa(gift.Receiver)).Err()
	if err != nil {
		log.Println(base.RankUpdateFail)
		return base.RankUpdateFail, err
	}
	//定时把rank写进mongo
	go timer()
	return base.Success, err
}

/*
定时写排行榜数据到redis
*/
func timer() {
	timeTickerChan := time.Tick(time.Minute * 5)
	for {
		rankSynToMongo()
		<-timeTickerChan
	}
}

/*
redis排行榜写进mongo
*/
func rankSynToMongo() {
	conn := connectRedis()
	defer conn.Close()
	values, err := conn.ZRevRange(ctx, base.RANK, 0, -1).Result()
	if err != nil {
		log.Println(base.RankWriteMongoFail)
		return
	}
	var rankItems = make([]base.RankItem, len(values))
	for i, value := range values {
		err = json.Unmarshal([]byte(value), &rankItems[i])
		rankItems[i].Rank = i + 1
		if err != nil {
			log.Println(base.RankWriteMongoFail)
			return
		}
	}
	session := connectMongo()
	defer session.Close()
	defer Reduce()
	db := session.DB(dbName)
	collection := db.C(rankCollection)
	for _, item := range rankItems {
		selector := bson.M{
			"receiver": item.UserId,
		}
		query := bson.M{
			"$setOnInsert": bson.M{
				"receiver": item.UserId,
				"rank":     item.Rank,
				"value":    item.Value,
			},
			"&set": bson.M{
				"rank":  item.Rank,
				"value": item.Value,
			},
		}
		_, err = collection.Upsert(selector, query)
		if err != nil {
			//TODO 是不是应该开启事务
			log.Println(base.RankWriteMongoFail)
			return
		}
	}
}

/*
获取送礼记录
*/
func GetGiftLog(receiver int, start int, end int) (giftLogs []base.Gift, msg string, err error) {
	//先从缓存拿，没有再读db，并写进缓存
	conn := connectRedis()
	defer conn.Close()
	key := strconv.Itoa(receiver)
	_, err = conn.Get(ctx, key).Result()
	if err == redis.Nil {
		//缓存中没有，从db中取 TODO 防击穿
		session := connectMongo()
		defer session.Close()
		defer Reduce()
		db := session.DB(dbName)
		collection := db.C(giftCollection)
		query := bson.M{
			"receiver": receiver,
		}
		var allGiftLogs []base.Gift
		err = collection.Find(query).All(&allGiftLogs)
		if err != nil {
			log.Println(base.GiftLogReadFail, err)
			return nil, base.GiftLogReadFail, err
		}
		giftLogs = allGiftLogs[start-1 : end]
		//取出来之后放到缓存里
		var giftsStr = make([][]byte, len(giftLogs))
		for i, g := range giftLogs {
			giftsStr[i], err = json.Marshal(g)
			if err != nil {
				log.Println(base.GiftLogLoadRedisFail, err)
				//写入缓存失败，但是不影响业务返回
				return giftLogs, base.Success, err
			}
		}
		_, err = conn.RPush(ctx, key, giftsStr).Result()
		if err != nil {
			log.Println(base.GiftLogLoadRedisFail, err)
			//写入缓存失败，但是不影响业务返回
			return giftLogs, base.Success, err
		}
	} else if err == nil {
		//缓存里存在
		var values []string
		values, err = conn.LRange(ctx, key, int64(start), int64(end)).Result()
		giftLogs = make([]base.Gift, len(values))
		for i, value := range values {
			err = json.Unmarshal([]byte(value), &giftLogs[i])
			if err != nil {
				log.Println(base.GiftLogReadFail, err)
				return nil, base.GiftLogReadFail, err
			}
			//fmt.Printf("GetGiftLog : sender:%d, receiver:%d, gift:%d, time:%s\n", giftLogs[i].Sender, giftLogs[i].Receiver, giftLogs[i].GiftType, giftLogs[i].Timestamp)
		}
	} else {
		log.Println(base.GiftLogReadFail, err)
		return nil, base.GiftLogReadFail, err
	}
	return giftLogs, base.Success, err
}

/*
获取排行榜数据
*/
func GetRank(start int, end int) (rankItems []base.RankItem, msg string, err error) {
	conn := connectRedis()
	defer conn.Close()
	//先从缓存拿，没有再读db，并写进缓存
	_, err = conn.Get(ctx, base.RANK).Result()
	if err == redis.Nil {
		//缓存中没有，从db中取 TODO 加锁，防击穿
		session := connectMongo()
		defer session.Close()
		defer Reduce()
		db := session.DB(dbName)
		collection := db.C(rankCollection)
		var allRankItems []base.RankItem
		err = collection.Find(nil).All(&allRankItems)
		if err != nil {
			log.Println(base.RankReadFail, err)
			return nil, base.RankReadFail, err
		}
		rankItems = allRankItems[start-1 : end]
		//取出来之后放到缓存里
		for _, rankItem := range rankItems {
			conn.ZIncrBy(ctx, base.RANK, float64(rankItem.Value), strconv.Itoa(rankItem.UserId))
		}
	} else if err == nil {
		//缓存里存在
		var result []redis.Z
		result, err = conn.ZRevRangeWithScores(ctx, base.RANK, int64(start), int64(end)).Result()
		if err != nil {
			log.Println(base.RankReadFail, err)
			return nil, base.RankReadFail, err
		}
		rankItems, err = createRankItems(result)
	} else {
		log.Println(base.RankReadFail, err)
		return nil, base.RankReadFail, err
	}
	return rankItems, base.Success, err
}

/*
构建排行项
*/
func createRankItems(values []redis.Z) (rankItems []base.RankItem, err error) {
	rankItems = make([]base.RankItem, len(values))
	for i := 0; i < len(values); i++ {
		key := values[i].Member
		var userId int
		userId, err = strconv.Atoi(key.(string))
		if err != nil {
			return nil, err
		}
		rankItems[i] = base.RankItem{Rank: i + 1, UserId: userId, Value: int(values[i].Score)}
	}
	return rankItems, nil
}

/*
启动的时候把排行榜加载到redis
*/
func LoadRedis() {
	conn := connectRedis()
	defer conn.Close()
	session := connectMongo()
	defer session.Close()
	defer Reduce()
	db := session.DB(dbName)
	collection := db.C(rankCollection)
	var allRankItems []base.RankItem
	err := collection.Find(nil).All(&allRankItems)
	if err != nil {
		log.Fatalln(base.RankReadFail, err)
	}
	//取出来之后放到缓存里
	for _, rankItem := range allRankItems {
		conn.ZIncrBy(ctx, base.RANK, float64(rankItem.Value), strconv.Itoa(rankItem.UserId))
	}
}
