package main

/**
服务
*/

import (
	"../handler"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/sendGift", handler.SendGift)
	http.HandleFunc("/rank", handler.Rank)
	http.HandleFunc("/log", handler.Log)
	println("启动成功!!!")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
	//runtime.GOMAXPROCS(10) //mongo最多10个协程访问
}
