package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jessevdk/go-flags"
	logviewer "github.com/y-akahori-ramen/gojobcoordinatortest/logViewer"
)

var opts struct {
	Addr      string `long:"addr" description:"サーバーアドレス (例)localhost:8000" required:"true"`
	LogDBAddr string `long:"logdb" description:"ログDBへの接続情報　(例)mongodb://fluentd:fluentdPassword@localhost:27017" required:"true"`
}

func main() {

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	logData, err := logviewer.NewMongoLogData(opts.LogDBAddr)
	if err != nil {
		os.Exit(1)
	}
	defer logData.Close()

	server, err := NewServer(logData)
	if err != nil {
		os.Exit(1)
	}

	err = http.ListenAndServe(opts.Addr, server.NewHTTPHandler())
	if err != nil {
		log.Fatal(err)
	}
}
