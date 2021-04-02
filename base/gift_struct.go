package base

import (
	"encoding/json"
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

func (g Gift) MarshalBinary() ([]byte, error) {
	return json.Marshal(g)
}

func (g Gift) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, g)
}

/*
排行榜项
*/
type RankItem struct {
	Rank   int
	UserId int
	Value  int
}
