package packet

import "time"

/*
返回客户端的送礼记录
*/
type GiftItem struct {
	Sender    uint64
	GiftType  int
	Timestamp time.Time
}

/*
返回客户端的排行榜项
*/
type RankItem struct {
	Rank   int
	UserId uint64
	Value  uint
}
