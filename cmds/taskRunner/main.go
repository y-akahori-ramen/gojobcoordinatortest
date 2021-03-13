package main

import (
	"log"
	"net/http"
)

func main() {
	server := &taskRunnerServer{}
	router := server.NewRouter()
	go func() {
		server.Run()
	}()
	log.Fatal(http.ListenAndServe(":8000", router))
}
