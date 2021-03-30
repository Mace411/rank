package main

/**
服务
*/

import (
	"log"
	"net/http"
	"runtime"
)
import "../handler"

func main() {
	http.HandleFunc("/sendGift", handler.SendGift)
	http.HandleFunc("/rank", handler.Rank)
	http.HandleFunc("/log", handler.Log)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
	runtime.GOMAXPROCS(10) //mongo最多10个协程访问
}
