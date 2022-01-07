// +build windows

package server

import (
	"os"
	"os/signal"
	"syscall"

	"git.code.oa.com/trpc-go/trpc-go/log"
)

// Serve 启动所有服务
func (s *Server) Serve() error {
	defer log.Sync()
	if len(s.services) == 0 {
		panic("service empty")
	}

	for _, service := range s.services {
		go func(s Service) {
			err := s.Serve()
			if err != nil {
				panic(err)
			}
		}(service)
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV)
	<-ch

	s.Close(nil)
	return nil
}
