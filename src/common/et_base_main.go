package common

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	CANCEL_DEAMON_ENV_KEY string = "__CANCEL_DAEMON__"
)

func getCurrPath() (ret string) {
	ret, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "filepath abs fail, will use /tmp to record running info, err=[%s]\n", err.Error())
		ret = "/tmp"
	}
	return ret
}

func redirectFd(name string) uintptr {
	dir := getCurrPath()
	fname := dir + "/" + name + ".run"
	file, _ := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
	return file.Fd()
}

func BaseMain(appMain func(command Command)) {
	rfd := redirectFd("eTunnel")
	var command Command
	command.parseCommand(os.Args)

	if _, found := syscall.Getenv(CANCEL_DEAMON_ENV_KEY); !found && *command.StartDaemon {
		syscall.Setenv(CANCEL_DEAMON_ENV_KEY, "")
		pa := syscall.ProcAttr{}
		pa.Dir, _ = os.Getwd()
		pa.Env = os.Environ()
		pa.Files = []uintptr{rfd, rfd, rfd}
		pid, err := syscall.ForkExec(os.Args[0], os.Args, &pa)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Start daemon fail, err=[%s]\n", err.Error())
			os.Exit(-1)
		} else {
			fmt.Fprintf(os.Stdout, "Start daemon succ, pid=[%v]\n", pid)
			os.Exit(0)
		}
	} else {
		if *command.StartDaemon {
			sid, err := syscall.Setsid()
			fmt.Fprintf(os.Stdout, "Setsid session_id=[%d] err=[%v]\n", sid, err)
		}
		appMain(command)
	}
}
