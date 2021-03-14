package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	var addr = flag.String("addr", "localhost:8000", "サーバーアドレス")
	var maxTaskNum = flag.Int("maxTaskNum", 2, "同時実行できる最大タスク数")
	flag.Parse()

	server := &taskRunnerServer{taskNumMax: *maxTaskNum}
	router := server.NewRouter()
	go func() {
		server.Run()
	}()

	fmt.Printf("サーバー起動します addr:%v 同時タスク実行数最大:%v\n", *addr, *maxTaskNum)

	log.Fatal(http.ListenAndServe(*addr, router))
}
