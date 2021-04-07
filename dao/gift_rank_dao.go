package dao

import (
	"../base"
	"../config"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"math"
	"strconv"
	"sync"
	"time"
)

type Lock struct {
	rankLock          *sync.RWMutex
	logLock           *sync.RWMutex
	openRankScheduler *sync.Mutex
}

var ctx = context.Background()

//是否已经开了排行榜存储到mongo的定时器
var openRankToMongoScheduler bool

//
var lock Lock

func init() {
	lock = Lock{
		rankLock:          new(sync.RWMutex),
		logLock:           new(sync.RWMutex),
		openRankScheduler: new(sync.Mutex),
	}
}

/*
添加送礼记录
*/
func AddSendGiftLog(gift base.Gift) (msg string, err error) {
	gift.Timestamp = time.Now()
	//先写db再写缓存
	session, err := connectMongo()
	if err != nil {
		return base.ServiceBusy, err
	}
	defer CloseMongo()
	db := session.DB(dbName)
	collection := db.C(giftCollection)
	err = collection.Insert(&gift)
	if err != nil {
		log.Println(base.SendGiftFailed, err)
		return base.SendGiftFailed, err
	}
	return base.Success, err
}

func DelRedisKey(key string) {
	redisConn := connectRedis()
	//defer redisConn.Close()
	redisConn.Del(ctx, key)
}

/*
更新排行榜
*/
func UpdateRank(gift base.Gift) (msg string, err error) {
	conn := connectRedis()
	//defer conn.Close()
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
	//定时把rank写进mongo,但是只能执行一次
	if !openRankToMongoScheduler {
		lock.openRankScheduler.Lock()
		if !openRankToMongoScheduler {
			openRankToMongoScheduler = true
			go scheduledUpdateRankToMongo()
			fmt.Printf("\n开启排行榜定时器之后的值:%t\n\n", openRankToMongoScheduler)
		}
		lock.openRankScheduler.Unlock()
	}
	return base.Success, err
}

/*
定时写排行榜数据到redis
*/
func scheduledUpdateRankToMongo() {
	timeTickerChan := time.Tick(time.Second * 5)
	for {
		rankSyncToMongo()
		<-timeTickerChan
	}
}

/*
redis排行榜写进mongo
*/
func rankSyncToMongo() {
	conn := connectRedis()
	//defer conn.Close()
	values, err := conn.ZRevRangeWithScores(ctx, base.RANK, 0, -1).Result()
	if err != nil {
		log.Println(base.RankWriteToMongoFail)
		return
	}
	if len(values) <= 0 {
		return
	}
	var rankItems = make([]base.RankItem, len(values))
	rankItems, err = createRankItems(values)
	session, err := connectMongo()
	if err != nil {
		log.Println(base.ServiceBusy, err)
		return
	}
	defer CloseMongo()
	db := session.DB(dbName)
	collection := db.C(rankCollection)
	for _, item := range rankItems {
		selector := bson.M{
			"userid": item.UserId,
		}
		query := bson.M{
			"$setOnInsert": bson.M{
				"userid": item.UserId,
			},
			"$set": bson.M{
				"rank":  item.Rank,
				"value": item.Value,
			},
		}
		_, err = collection.Upsert(selector, query)
		if err != nil {
			//TODO 是不是应该开启事务
			log.Println(base.RankWriteToMongoFail, err)
			return
		}
	}
}

/*
获取送礼记录 TODO 校验开始和结尾的参数，控制获取的数量
*/
func GetGiftLog(receiver int, start int, end int) (giftLogs []base.Gift, msg string, err error) {
	//先从缓存拿，没有再读db，并写进缓存
	conn := connectRedis()
	//defer conn.Close()
	key := strconv.Itoa(receiver)
	var count int64
	count, err = conn.Exists(ctx, key).Result()
	if count <= 0 {
		lock.logLock.Lock()
		count, err = conn.Exists(ctx, key).Result()
		if count <= 0 {
			//缓存中没有，从db中取
			var session *mgo.Session
			session, err = connectMongo()
			if err != nil {
				return nil, base.ServiceBusy, err
			}
			defer CloseMongo()
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
			//返回给客户端的部分
			var min = int(math.Min(float64(len(allGiftLogs)), float64(end)))
			fmt.Printf("GetGiftLog.min:%d", min)
			giftLogs = allGiftLogs[start:min]
			//取出来之后所有的记录放到缓存里
			var giftsStr = make([]interface{}, len(allGiftLogs))
			for i, g := range allGiftLogs {
				giftsStr[i], err = json.Marshal(g)
				if err != nil {
					log.Println(base.GiftLogLoadToRedisFail, err)
					//写入缓存失败，但是不影响业务返回
					return giftLogs, base.Success, nil
				}
			}
			_, err = conn.RPush(ctx, key, giftsStr...).Result()
			if err != nil {
				log.Println(base.GiftLogLoadToRedisFail, err)
				return giftLogs, base.Success, nil
			}
		}
		lock.logLock.Unlock()
	} else if err != nil {
		log.Println(base.GiftLogReadFail, err)
		return nil, base.GiftLogReadFail, err
	}
	if count > 0 {
		//缓存里存在 TODO 需要考虑并发时，缓存被删了
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
	}
	return giftLogs, base.Success, nil
}

/*
获取排行榜数据 TODO 校验开始和结尾参数，控制获取的数量
*/
func GetRank(start int, end int) (rankItems []base.RankItem, msg string, err error) {
	conn := connectRedis()
	//defer conn.Close()
	//先从缓存拿，没有再读db，并写进缓存
	var count int64
	count, err = conn.Exists(ctx, base.RANK).Result()
	if count <= 0 {
		lock.rankLock.Lock()
		count, err = conn.Exists(ctx, base.RANK).Result()
		if count <= 0 {
			//缓存中没有，从db中取
			var session *mgo.Session
			session, err = connectMongo()
			if err != nil {
				return nil, base.ServiceBusy, err
			}
			db := session.DB(dbName)
			collection := db.C(rankCollection)
			var allRankItems []base.RankItem
			err = collection.Find(nil).All(&allRankItems)
			if err != nil {
				log.Println(base.RankReadFail, err)
				return nil, base.RankReadFail, err
			}
			//返回给客户端的部分
			var min = int(math.Min(float64(len(rankItems)), float64(end)))
			fmt.Printf("GetRank.min:%d", min)
			rankItems = allRankItems[start:min]
			//取出来之后放到缓存里
			var z = make([]*redis.Z, len(allRankItems))
			for i, rankItem := range allRankItems {
				z[i] = &redis.Z{
					Member: strconv.Itoa(rankItem.UserId),
					Score:  float64(rankItem.Value),
				}
			}
			_, err = conn.ZAdd(ctx, base.RANK, z...).Result()
			if err != nil {
				//写入缓存失败不应该影响业务
				log.Println(base.RankLoadToRedisFail, err)
				return rankItems, base.RankLoadToRedisFail, nil
			}
		}
		lock.rankLock.Unlock()
	} else if err != nil {
		log.Println(base.RankReadFail, err)
		return nil, base.RankReadFail, err
	}
	if count > 0 {
		//缓存里存在 TODO 需要考虑缓存被删的情况
		var result []redis.Z
		result, err = conn.ZRevRangeWithScores(ctx, base.RANK, int64(start), int64(end)).Result()
		if err != nil {
			log.Println(base.RankReadFail, err)
			return nil, base.RankReadFail, err
		}
		rankItems, err = createRankItems(result)
		if err != nil {
			log.Println(base.RankReadFail, err)
			return nil, base.RankReadFail, err
		}
	}
	return rankItems, base.Success, nil
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
func LoadRankToRedis() {
	conn := connectRedis()
	//defer conn.Close()
	session, err := connectMongo()
	if err != nil {
		//讲道理这里不会被return
		return
	}
	defer CloseMongo()
	db := session.DB(dbName)
	collection := db.C(rankCollection)
	var allRankItems []base.RankItem
	err = collection.Find(nil).All(&allRankItems)
	if err != nil {
		log.Fatalln(base.RankReadFail, err)
	}
	//取出来之后放到缓存里
	var z = make([]*redis.Z, len(allRankItems))
	for i, rankItem := range allRankItems {
		z[i] = &redis.Z{
			Member: strconv.Itoa(rankItem.UserId),
			Score:  float64(rankItem.Value),
		}
	}
	_, err = conn.ZAdd(ctx, base.RANK, z...).Result()
	if err != nil {
		//写入缓存失败不应该影响业务
		log.Fatalln(base.RankLoadToRedisFail, err)
	}
}
