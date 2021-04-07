package main

/**
服务
*/

import (
	"../dao"
	"../handler"
	"log"
	"net/http"
)

func main() {
	//加载排行榜到redis
	dao.LoadRankToRedis()
	//注册handler
	http.HandleFunc("/sendGift", handler.SendGift)
	http.HandleFunc("/rank", handler.Rank)
	http.HandleFunc("/log", handler.Log)
	println("启动成功!!!")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
