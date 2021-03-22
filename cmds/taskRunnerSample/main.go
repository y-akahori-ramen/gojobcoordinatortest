package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

func main() {
	var addr = flag.String("addr", "localhost:8000", "サーバーアドレス")
	var maxTaskNum = flag.Uint("maxTaskNum", 2, "同時実行できる最大タスク数")
	flag.Parse()

	server := gojobcoordinatortest.NewTaskRunnerServer(*maxTaskNum)
	server.AddFactory(ProcNameWait, newWaitTask)
	server.AddFactory(ProcNameEcho, newEchoTask)
	router := server.NewHTTPHandler()
	go func() {
		server.Run()
	}()

	fmt.Printf("サーバー起動します addr:%v 同時タスク実行数最大:%v\n", *addr, *maxTaskNum)

	log.Fatal(http.ListenAndServe(*addr, router))
}
