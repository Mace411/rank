package handler

import (
	"../base"
	"../config"
	"../dao"
	"../packet"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

/*
服务handler
*/

func errorMsg(desc string, err error, w http.ResponseWriter) {
	log.Println(desc, err)
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
		return
	}
	//fmt.Printf("sender:%d, receiver:%d, gift:%d\n", gift.Sender, gift.Receiver, gift.GiftType)
	value := config.GiftConfigMap[gift.GiftType]
	if value <= 0 {
		errorMsg(base.ParamError, nil, w)
		return
	}
	//送礼业务
	//往redis里添加送花记录
	msg, err := dao.AddSendGiftLog(gift)
	if err != nil {
		errorMsg(msg, err, w)
		return
	}
	dao.DelRedisKey(strconv.Itoa(gift.Receiver))
	//往redis里更新排行榜
	msg, err = dao.UpdateRank(gift)
	if err != nil {
		errorMsg(msg, err, w)
		return
	}
	if err = json.NewEncoder(w).Encode(msg); err != nil {
		log.Println(err)
	}
}

/*
排行，用get，返回数据用json
*/
func Rank(w http.ResponseWriter, r *http.Request) {
	rankItems, msg, err := dao.GetRank(0, 99)
	if err != nil {
		errorMsg(msg, err, w)
		return
	}
	json.NewEncoder(w).Encode(rankItems)
}

/*
流水，用get，返回数据用json
*/
func Log(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	receiverId, err := strconv.Atoi(query.Get("receiverId"))
	if err != nil {
		errorMsg(base.ParamError, err, w)
		return
	}
	giftLogs, msg, err := dao.GetGiftLog(receiverId, 0, 99)
	if err != nil {
		errorMsg(msg, err, w)
		return
	}
	giftLogsResp := make([]packet.GiftItem, len(giftLogs))
	for i, giftLog := range giftLogs {
		giftLogsResp[i] = packet.GiftItem{Sender: giftLog.Sender, GiftType: giftLog.GiftType, Timestamp: giftLog.Timestamp}
	}
	//TODO 按照时间近到远排序
	json.NewEncoder(w).Encode(giftLogsResp)
}
