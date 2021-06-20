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
	var addr = flag.String("addr", "localhost:8080", "サーバーアドレス")
	flag.Parse()

	cod := gojobcoordinatortest.NewCoordinator(gojobcoordinatortest.CoordinatorConfig{})
	server := gojobcoordinatortest.NewCoordinatorServer(cod)
	fmt.Println("サーバー起動します:", *addr)

	go func() {
		server.Run(context.Background())
	}()

	log.Fatal(http.ListenAndServe(*addr, server.NewHTTPHandler()))
}
