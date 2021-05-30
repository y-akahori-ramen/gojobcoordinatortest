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
	var nsqdUri = flag.String("nsqdUri", "", "NSQにタスクのログを送る場合NSQDのアドレス")
	flag.Parse()

	var server *gojobcoordinatortest.TaskRunnerServer
	if *nsqdUri != "" {
		var err error
		server, err = gojobcoordinatortest.NewTaskRunnerServerNSQ(*maxTaskNum, *nsqdUri)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		server = gojobcoordinatortest.NewTaskRunnerServer(*maxTaskNum)
	}
	server.AddFactory(ProcNameWait, newWaitTask)
	server.AddFactory(ProcNameEcho, newEchoTask)
	router := server.NewHTTPHandler()
	go func() {
		server.Run()
	}()

	fmt.Printf("サーバー起動します addr:%v 同時タスク実行数最大:%v\n", *addr, *maxTaskNum)

	log.Fatal(http.ListenAndServe(*addr, router))
}
