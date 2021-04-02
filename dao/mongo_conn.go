package dao

import (
	"gopkg.in/mgo.v2"
	"time"
)

const (
	url                          = "mongodb://localhost:27017"
	dbName                       = "gift"
	timeout        time.Duration = 10
	num            int           = 10
	userCollection               = "user"
	giftCollection               = "gift"
	rankCollection               = "rank"
	maxConnect                   = 10
)

var globalMgoSession *mgo.Session

/*
控制mongo的访问数量
*/
var sem = make(chan int, maxConnect)

func session() *mgo.Session {
	var err error
	globalMgoSession, err = mgo.DialWithTimeout(url, timeout)
	if err != nil {
		panic(err)
	}
	globalMgoSession.SetMode(mgo.Monotonic, true)
	globalMgoSession.SetPoolLimit(num)
	return globalMgoSession
}

/*
获取mongo连接
*/
func connectMongo() *mgo.Session {
	sem <- 1
	if globalMgoSession == nil {
		globalMgoSession = session()
	}
	return globalMgoSession.Clone()
}

/*
用完mongo之后返还
*/
func Reduce() {
	<-sem
}
