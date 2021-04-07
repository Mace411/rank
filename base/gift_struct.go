package base

import (
	"time"
)

/*
送礼
*/
type Gift struct {
	Sender    int
	Receiver  int
	GiftType  int
	Timestamp time.Time
}

/*
排行榜项
*/
type RankItem struct {
	Rank   int
	UserId int
	Value  int
}
