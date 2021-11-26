package process

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nearmeng/mango-go/plugin/log"
)

const _killProcessWaitTime = 10 * time.Microsecond

// KillPre 根据entityid杀掉之前的进程, 并写入pid文件.
func KillPre(entityid string) error {
	// 1. 读取文件获取pid
	if err := KillProcess(entityid); err != nil {
		return err
	}

	// 2. 写入新的
	return WritePidFile(entityid)
}

// KillProcess 杀掉进程.
func KillProcess(entityid string) error {
	filePath := fmt.Sprintf("/tmp/%s.pid", entityid)
	content, err := ioutil.ReadFile(filePath)
	if err == nil {
		pidStr := strings.ReplaceAll(string(content), "\n", "")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return fmt.Errorf("parse pidstr=%s fail", pidStr)
		}

		// 看之前的进程还在不在, 如果还在, 那么就直接kill
		p1, _ := os.FindProcess(pid)
		if err := p1.Kill(); err == nil {
			time.Sleep(_killProcessWaitTime)
			log.Info("kill pre pid=%d", pid)
		}
	}
	return nil
}

// WritePidFile 写入pid文件.
func WritePidFile(entityid string) error {
	filePath := fmt.Sprintf("/tmp/%s.pid", entityid)
	curPid := os.Getpid()
	log.Info("cur pid=%d", curPid)
	curPidStr := strconv.Itoa(curPid)
	bs := []byte(curPidStr)
	if err := ioutil.WriteFile(filePath, bs, 0644); err != nil { // nolint
		return fmt.Errorf("write pid file fail")
	}
	return nil
}

// Daemon 进程改为守护方式执行.
func Daemon(nochdir, noclose int) (int, error) {
	// already a daemon
	if syscall.Getppid() == 1 {
		/* Change the file mode mask */
		syscall.Umask(0)

		if nochdir == 0 {
			_ = os.Chdir("/")
		}

		return 0, nil
	}

	curFileNum := 3
	maxFileNum := 6
	files := make([]*os.File, curFileNum, maxFileNum)
	if noclose == 0 {
		nullDev, err := os.OpenFile("/dev/null", 0, 0)
		if err != nil {
			return 1, err
		}
		files[0], files[1], files[2] = nullDev, nullDev, nullDev
	} else {
		files[0], files[1], files[2] = os.Stdin, os.Stdout, os.Stderr
	}

	dir, _ := os.Getwd()
	sysattrs := syscall.SysProcAttr{Setsid: true}
	attrs := os.ProcAttr{Dir: dir, Env: os.Environ(), Files: files, Sys: &sysattrs}

	proc, err := os.StartProcess(os.Args[0], os.Args, &attrs)
	if err != nil {
		return -1, fmt.Errorf("can't create process=%s err:%w", os.Args[0], err)
	}
	_ = proc.Release()
	os.Exit(0) // nolint

	return 0, nil
}
