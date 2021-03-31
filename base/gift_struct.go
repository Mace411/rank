package base

import "time"

/*
送礼
*/
type Gift struct {
	Sender    uint64
	Receiver  uint64
	GiftType  int
	Timestamp time.Time
}
