package redisdao

import (
	"../base"
	"../config"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

/*
添加送礼记录
*/
func AddSendGiftLog(gift base.Gift) (err error) {
	conn := connect()
	defer conn.Close()
	//收礼者id为key
	gift.Timestamp = time.Now()
	giftJson, _ := json.Marshal(gift)
	_, err = conn.Do("RPUSH", gift.Receiver, giftJson)
	if err != nil {
		log.Println("送礼记录失败", err)
	}
	return err
}

func UpdateRank(gift base.Gift) (err error) {
	conn := connect()
	defer conn.Close()
	value := config.GiftConfigMap[gift.GiftType]
	if value <= 0 {
		log.Println("礼物类型不存在")
		return err
	}
	_, err = conn.Do("ZINCRBY", base.RANK, value, gift.Receiver)
	if err != nil {
		log.Println("排行榜更新失败")
	}
	return err
}

/*
获取送礼记录
*/
func GetGiftLog(receiver uint64) (giftLogs []base.Gift, err error) {
	conn := connect()
	defer conn.Close()
	values, err := redis.Values(conn.Do("LRANGE", receiver, 0, -1))
	giftLogs = make([]base.Gift, len(values))
	for i, value := range values {
		json.Unmarshal(value.([]byte), &giftLogs[i])
		fmt.Printf("sender:%d, receiver:%d, gift:%d, time:%s\n", giftLogs[i].Sender, giftLogs[i].Receiver, giftLogs[i].GiftType, giftLogs[i].Timestamp)
	}
	return
}

func GetRank() (values []interface{}, err error) {
	conn := connect()
	defer conn.Close()
	values, err = redis.Values(conn.Do("ZREVRANGE", base.RANK, 0, -1, base.WithScores))
	if err != nil {
		log.Println("获取排行榜失败")
		return nil, err
	}
	return values, err
}
