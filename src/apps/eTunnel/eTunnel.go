package main

import (
	"common"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"proxy"
)

func app_main() {
	if err := common.ParseCommandAndFile(); err != nil {
		fmt.Fprintf(os.Stderr, "ParseCommandAndFile fail, err=[%v]\n", err)
		os.Exit(-1)
	}
	go http.ListenAndServe("0.0.0.0:8060", nil)
	http.ListenAndServe("0.0.0.0:8459", proxy.NewProxyServer())
}

func main() {
	common.BaseMain(app_main)
}
