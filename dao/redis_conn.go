package dao

import (
	"github.com/go-redis/redis"
)

const (
	//network = "tcp"
	address = "127.0.0.1:6379"
)

//TODO 连接池
func connectRedis() *redis.Client {
	//获取连接
	conn := redis.NewClient(&redis.Options{
		Addr: address,
	})
	return conn
}
