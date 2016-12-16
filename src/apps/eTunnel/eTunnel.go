package main

import (
	"fmt"
	"net/http"
	//_ "net/http/pprof"
	"common"
	"os"
	"proxy"
)

func app_main() {
	if err := common.ParseCommandAndFile(); err != nil {
		fmt.Fprintf(os.Stderr, "ParseCommandAndFile fail, err=[%v]\n", err)
		os.Exit(-1)
	}
	http.ListenAndServe("0.0.0.0:8459", proxy.NewProxyServer())
}

func main() {
	common.BaseMain(app_main)
}
