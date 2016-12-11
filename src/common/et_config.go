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
	LogConfigFile     string `check:"StringNotEmpty"`
	ServerBindAddress string `check:"StringNotEmpty"`
	Salt              string `check:"StringNotEmpty"`
}

type server struct {
	ConnectionTimeoutSec int64 `check:"IntGTZero"`
}

type client struct {
	ClientBindAddress string `check:"StringNotEmpty"`
	KeepAliveTimeSec  int64  `check:"IntGTZero"`
}

type command struct {
	ConfigFile   *string
	LogConfFile  *string
	PrintVersion *bool
	StartDaemon  *bool
	Type         *string
}

////////////////////////////////////////////////////////////////////////////////////////////////////

func (self *command) parseCommand(args []string) {
	flagset := flag.NewFlagSet(MY_NAME, flag.ExitOnError)
	self.ConfigFile = flagset.String("config", "./etc/eTunnel.conf", "Path to config file")
	self.LogConfFile = flagset.String("logconf", "", "Path to config file")
	self.PrintVersion = flagset.Bool("version", false, "Print etunnel version")
	self.StartDaemon = flagset.Bool("daemon", false, "Start daemon server")
	self.Type = flagset.String("type", "", "eTunnel type server/client")
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
	var command command
	command.parseCommand(os.Args)

	_, err := toml.DecodeFile(*command.ConfigFile, &G)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config file parse fail, err=[%s] file=[%s]\n", err.Error(), *command.ConfigFile)
		os.Exit(-1)
	}

	err = G.check()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config check fail, err=[%s]\n", err.Error())
		os.Exit(-1)
	}

	if *command.LogConfFile != "" {
		G.Basic.LogConfigFile = *command.LogConfFile
	}
	logger, err := log.LoggerFromConfigAsFile(G.Basic.LogConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seelog LoggerFromConfigAsFile fail, err=[%s]\n", err.Error())
		os.Exit(-1)
	}
	log.ReplaceLogger(logger)

	if *command.Type != "server" &&
		*command.Type != "client" {
		fmt.Fprintf(os.Stderr, "type must be 'server' or 'client'\n")
		os.Exit(-1)
	}

	log.Infof("parse config succ, %s", G.String())
	return err
}
