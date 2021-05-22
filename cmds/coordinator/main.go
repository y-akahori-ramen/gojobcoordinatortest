package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	var addr = flag.String("addr", "localhost:8080", "サーバーアドレス")
	flag.Parse()

	server := coordinatorServer{}
	router := server.newRouter()

	fmt.Println("サーバー起動します:", *addr)

	// 一定間隔で接続しているTaskRunnerの生存を確認する
	go func() {
		ticker := time.NewTicker(time.Second * 30)
		for range ticker.C {
			server.removeDeadTaskRunners()
		}
	}()

	log.Fatal(http.ListenAndServe(*addr, router))
}
