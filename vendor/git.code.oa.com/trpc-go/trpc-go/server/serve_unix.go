//go:build !windows
// +build !windows

package server

import (
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/transport"
)

// DefaultServerCloseSIG close信号量
var DefaultServerCloseSIG = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV}

// DefaultServerGracefulSIG 热重启信号量
var DefaultServerGracefulSIG = syscall.SIGUSR2

// EnvGraceRestartStr 是否热重启环境变量
const EnvGraceRestartStr = "TRPC_IS_GRACEFUL=1"

// Serve 启动所有服务
func (s *Server) Serve() error {
	defer log.Sync()
	if len(s.services) == 0 {
		panic("service empty")
	}

	ch := make(chan os.Signal)
	var failedServices sync.Map
	var err error
	for name, service := range s.services {
		go func(n string, srv Service) {
			if e := srv.Serve(); e != nil {
				err = e
				failedServices.Store(n, srv)
				time.Sleep(time.Millisecond * 300)
				ch <- syscall.SIGTERM
			}
		}(name, service)
	}

	/*
		signal.Notify(ch, append(DefaultServerCloseSIG, DefaultServerGracefulSIG)...)

		sig := <-ch
		// 热重启单独处理
		if sig == DefaultServerGracefulSIG {
			if _, err := s.StartNewProcess(); err != nil {
				panic(err)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), MaxCloseWaitTime)
		defer cancel()

		var wg sync.WaitGroup
		for name, service := range s.services {
			if _, ok := failedServices.Load(name); ok {
				continue
			}

			wg.Add(1)
			go func(srv Service) {
				defer wg.Done()

				c := make(chan struct{}, 1)
				go srv.Close(c)
				select {
				case <-c:
				case <-ctx.Done():
				}
			}(service)
		}

		wg.Wait()
		if err != nil {
			panic(err)
		}
	*/
	return nil
}

// StartNewProcess 启动新进程
func (s *Server) StartNewProcess(args ...string) (uintptr, error) {
	pid := os.Getpid()
	log.Infof("process: %d, received graceful restart signal, so restart the process", pid)

	// pass tcp listeners' Fds and udp conn's Fds
	listenersFds := transport.GetListenersFds()

	files := []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}

	os.Setenv(transport.EnvGraceRestart, "1")
	os.Setenv(transport.EnvGraceFirstFd, strconv.Itoa(len(files)))
	os.Setenv(transport.EnvGraceRestartFdNum, strconv.Itoa(len(listenersFds)))

	files = append(files, prepareListenFds(listenersFds)...)

	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: files,
	}

	os.Args = append(os.Args, args...)
	childPID, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.Errorf("process: %d, failed to forkexec with err: %s", pid, err.Error())
		return 0, err
	}

	for _, f := range listenersFds {
		f.File.Close()
	}
	return uintptr(childPID), nil
}

func prepareListenFds(fds []*transport.ListenFd) []uintptr {
	files := make([]uintptr, 0, len(fds))
	for _, fd := range fds {
		files = append(files, fd.Fd)
	}
	return files
}
