package handler

import (
	"../base"
	"../config"
	"../packet"
	"../redisdao"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net/http"
	"strconv"
)

/*
服务handler
*/

func errorMsg(desc string, err error, w http.ResponseWriter) {
	log.Println(err)
	w.Write([]byte(desc))
}

/*
送花，使用json传送数据
*/
func SendGift(w http.ResponseWriter, r *http.Request) {
	var gift base.Gift
	if err := json.NewDecoder(r.Body).Decode(&gift); err != nil {
		r.Body.Close()
		errorMsg(base.ParamError, err, w)

	}
	fmt.Printf("sender:%d, receiver:%d, gift:%d\n", gift.Sender, gift.Receiver, gift.GiftType)
	value := config.GiftConfigMap[gift.GiftType]
	if value <= 0 {
		// TODO
	}
	var result string
	//送礼业务
	//往redis里添加送花记录
	err := redisdao.AddSendGiftLog(gift)
	if err != nil {
		// TODO
		return
	}
	//往redis里更新排行榜
	err = redisdao.UpdateRank(gift)
	if err != nil {
		//TODO
		return
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Println(err)
	}
}

/*
排行，用get，返回数据用josn
*/
func Rank(w http.ResponseWriter, r *http.Request) {
	rankValues, err := redisdao.GetRank()
	if err != nil {
		log.Println("获取排行榜数据失败")
		//TODO
	}
	rankItems, err := createRankItems(rankValues)
	if err != nil {
		//TODO
		return
	}
	json.NewEncoder(w).Encode(rankItems)
}

/*
构建排行项
*/
func createRankItems(values []interface{}) (rankItems []packet.RankItem, err error) {
	if len(values)%2 != 0 {
		return nil, errors.New("expects even number of values result")
	}
	rankItems = make([]packet.RankItem, len(values)/2)
	var index int
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].([]byte)
		if !ok {
			return nil, errors.New("redigo: IntMap key not a bulk string value")
		}
		value, err := redis.Int(values[i+1], nil)
		if err != nil {
			return nil, err
		}
		userId, _ := strconv.Atoi(string(key))
		rankItems[index] = packet.RankItem{Rank: index + 1, UserId: uint64(userId), Value: uint(value)}
		index++
	}
	return rankItems, nil
}

/*
流水，用get，返回数据用json
*/
func Log(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	receiverId, err := strconv.Atoi(query.Get("receiverId"))
	if err != nil {
		log.Println("收礼者id有误")
	}
	giftLogs, _ := redisdao.GetGiftLog(uint64(receiverId))
	giftLogsResp := make([]packet.GiftItem, len(giftLogs))
	json.NewEncoder(w).Encode(giftLogsResp)
}
