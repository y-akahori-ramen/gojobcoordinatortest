package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	var addr = flag.String("addr", "localhost:8080", "サーバーアドレス")
	flag.Parse()

	server := coordinatorServer{}
	router := server.newRouter()

	fmt.Println("サーバー起動します:", *addr)

	log.Fatal(http.ListenAndServe(*addr, router))
}
