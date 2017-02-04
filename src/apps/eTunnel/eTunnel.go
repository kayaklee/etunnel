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

	switch *common.C.Type {
	case "server":
		go http.ListenAndServe(common.G.Server.DebugBindAddress, nil)
		http.ListenAndServe(
			common.G.Server.BindAddress,
			proxy.NewProxyServer())
	case "client":
		go http.ListenAndServe(common.G.Client.DebugBindAddress, nil)
		proxy.NewClientServer(
			common.G.Client.BindAddress,
			common.G.Client.ServerAddress,
			*common.C.Dest).Start()
	}
}

func main() {
	common.BaseMain(app_main)
}
