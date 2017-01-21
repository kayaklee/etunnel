package main

import (
	"common"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"proxy"
)

func app_main(command common.Command) {
	if err := common.ParseCommandAndFile(); err != nil {
		fmt.Fprintf(os.Stderr, "ParseCommandAndFile fail, err=[%v]\n", err)
		os.Exit(-1)
	}
	go http.ListenAndServe("0.0.0.0:8060", nil)

	switch *command.Type {
	case "server":
		http.ListenAndServe(
			common.G.Basic.ServerBindAddress,
			proxy.NewProxyServer())
	case "client":
		proxy.NewClientServer(
			common.G.Client.ClientBindAddress,
			common.G.Basic.ServerBindAddress,
			*command.Addr).Start()
	}
}

func main() {
	common.BaseMain(app_main)
}
