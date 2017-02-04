package common

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	log "third/seelog"
	toml "third/toml"
)

var G etConfig
var C etCommand

const (
	MY_NAME string = "eTunnel"
)

type etConfig struct {
	Basic  basic  `check:"Struct"`
	Server server `check:"Struct"`
	Client client `check:"Struct"`
}

var (
	author     string = "Please build etunnel use build.sh"
	githash    string = "Please build etunnel use build.sh"
	buildstamp string = "Please build etunnel use build.sh"
	goversion  string = runtime.Version()
)

type basic struct {
}

type server struct {
	LogConfigFile        string `check:"StringNotEmpty"`
	BindAddress          string `check:"StringNotEmpty"`
	DebugBindAddress     string `check:"StringNotEmpty"`
	ConnectionTimeoutSec int64  `check:"IntGTZero"`
	KeepAliveTimeSec     int64  `check:"IntGTZero"`
	PrivateKeyFilePath   string `check:"StringNotEmpty"`
}

type client struct {
	LogConfigFile     string `check:"StringNotEmpty"`
	BindAddress       string `check:"StringNotEmpty"`
	DebugBindAddress  string `check:"StringNotEmpty"`
	ServerAddress     string `check:"StringNotEmpty"`
	PublicKeyFilePath string `check:"StringNotEmpty"`
}

type etCommand struct {
	ConfigFile   *string
	LogConfFile  *string
	PrintVersion *bool
	Foreground   *bool
	Type         *string
	Dest         *string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func (self *etCommand) parseCommand(args []string) {
	flagset := flag.NewFlagSet(MY_NAME, flag.ExitOnError)
	self.ConfigFile = flagset.String("config", "./etc/eTunnel.conf", "Path to config file")
	self.LogConfFile = flagset.String("logconf", "", "Path to config file")
	self.PrintVersion = flagset.Bool("version", false, "Print etunnel version")
	self.Foreground = flagset.Bool("fg", false, "Start server in foreground")
	self.Type = flagset.String("type", "", "eTunnel type server/client")
	self.Dest = flagset.String("dest", "", "eTunnel destination address")
	flagset.Parse(args[1:])

	if *self.PrintVersion {
		fmt.Fprintf(os.Stdout, "%s\n", MY_NAME)
		fmt.Fprintf(os.Stdout, "Author:           %s\n", author)
		fmt.Fprintf(os.Stdout, "Git Commit Hash:  %s\n", githash)
		fmt.Fprintf(os.Stdout, "Build Time:       %s\n", buildstamp)
		fmt.Fprintf(os.Stdout, "Go Version:       %s\n", goversion)
		os.Exit(0)
	}
}

func (self etConfig) check() error {
	return configCheckStruct(MY_NAME, self)
}

func (self etConfig) String() string {
	return configStringStruct(MY_NAME, self)
}

func ParseCommandAndFile() error {
	C.parseCommand(os.Args)

	if *C.Type != "server" &&
		*C.Type != "client" {
		fmt.Fprintf(os.Stderr, "type must be 'server' or 'client'\n")
		os.Exit(-1)
	}

	if *C.Type == "client" && *C.Dest == "" {
		fmt.Fprintf(os.Stderr, "addr can not be empty while type is client\n")
		os.Exit(-1)
	}

	_, err := toml.DecodeFile(*C.ConfigFile, &G)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config file parse fail, err=[%s] file=[%s]\n", err.Error(), *C.ConfigFile)
		os.Exit(-1)
	}

	err = G.check()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config check fail, err=[%s]\n", err.Error())
		os.Exit(-1)
	}

	var log_config_file string
	if *C.Type == "server" {
		if *C.LogConfFile != "" {
			G.Server.LogConfigFile = *C.LogConfFile
		}
		log_config_file = G.Server.LogConfigFile
	}
	if *C.Type == "client" {
		if *C.LogConfFile != "" {
			G.Client.LogConfigFile = *C.LogConfFile
		}
		log_config_file = G.Client.LogConfigFile
	}
	logger, err := log.LoggerFromConfigAsFile(log_config_file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seelog LoggerFromConfigAsFile fail, err=[%s]\n", err.Error())
		os.Exit(-1)
	}
	log.ReplaceLogger(logger)

	log.Infof("parse config succ, %s", G.String())
	return err
}
