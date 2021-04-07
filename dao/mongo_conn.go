package dao

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"sync/atomic"
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

type MongoConn struct {
	globalMgoSession *mgo.Session
	sem              int32
}

var mongoConn MongoConn

func session() MongoConn {
	globalMgoSession, err := mgo.Dial(url)
	if err != nil {
		panic(err)
	}
	mongoConn = MongoConn{globalMgoSession: globalMgoSession, sem: 0}
	return mongoConn
}

/*
获取mongo连接
*/
func connectMongo() (*mgo.Session, error) {
	if mongoConn.globalMgoSession == nil {
		mongoConn = session()
	}
	if mongoConn.sem >= maxConnect {
		return nil, fmt.Errorf("mongo的连接数达到上限")
	}
	atomic.AddInt32(&mongoConn.sem, 1)
	//fmt.Printf("获取之后的值:%d, 时间:%s\n", newInt, time.Now())
	return mongoConn.globalMgoSession, nil
}

/*
用完mongo之后返还
*/
func CloseMongo() {
	atomic.AddInt32(&mongoConn.sem, -1)
	//fmt.Printf("返还之后的值:%d, 时间:%s\n",newInt, time.Now())
}
