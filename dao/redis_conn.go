package dao

import (
	"github.com/go-redis/redis"
	"time"
)

const (
	//network = "tcp"
	address      = "127.0.0.1:6379"
	poolSize     = 24
	minIdleConns = 10
)

var redisConn *redis.Client

func init() {
	initConn()
}

func initConn() {
	//获取连接
	redisConn = redis.NewClient(&redis.Options{
		Addr:         address,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second, //读超时，默认3秒， -1表示取消读超时
		WriteTimeout: 3 * time.Second, //写超时，默认等于读超时
		PoolTimeout:  4 * time.Second, //当所有连接都处在繁忙状态时，客户端等待可用连接的最大等待时长，默认为读超时+1秒。
	})
}

func connectRedis() *redis.Client {
	return redisConn
}
