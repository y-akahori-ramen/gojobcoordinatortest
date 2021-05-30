package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

func main() {
	var addr = flag.String("addr", "localhost:8080", "サーバーアドレス")
	var nsqdUri = flag.String("nsqdUri", "", "NSQにジョブのログを送る場合NSQDのアドレス")
	flag.Parse()

	var jobWriter io.Writer
	if *nsqdUri != "" {
		var err error
		jobWriter, err = gojobcoordinatortest.NewNSQWriter(*nsqdUri, gojobcoordinatortest.JobTopicName, log.Default().Writer())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		jobWriter = log.Default().Writer()
	}

	server := coordinatorServer{jobWriter: jobWriter}
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
