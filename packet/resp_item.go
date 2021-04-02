package packet

import "time"

/*
返回客户端的送礼记录
*/
type GiftItem struct {
	Sender    int
	GiftType  int
	Timestamp time.Time
}
