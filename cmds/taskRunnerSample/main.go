package main

import (
	"context"
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

	runner := gojobcoordinatortest.NewTaskRunner(gojobcoordinatortest.TaskRunnerConfig{TaskNumMax: *maxTaskNum})
	runner.AddFactory(ProcNameWait, newWaitTask)
	runner.AddFactory(ProcNameEcho, newEchoTask)

	server := gojobcoordinatortest.NewTaskRunnerServer(runner)
	router := server.NewHTTPHandler()
	go func() {
		server.Run(context.Background())
	}()

	fmt.Printf("サーバー起動します addr:%v 同時タスク実行数最大:%v\n", *addr, *maxTaskNum)

	log.Fatal(http.ListenAndServe(*addr, router))
}
