package trpc

import (
	trpc "git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/server"
)

type TrpcServer struct {
	s *server.Server
}

func NewTrpcServer(config *TrpcConfig) *TrpcServer {
	trpc.ServerConfigPath = config.ConfigPath

	return &TrpcServer{
		s: trpc.NewServer(),
	}
}

func (ts *TrpcServer) Init() error {
	return nil
}

func (ts *TrpcServer) Mainloop() {
	ts.s.Serve()

}

func (ts *TrpcServer) UnInit() error {
	ts.s.Close(nil)
	return nil
}

func (ts *TrpcServer) GetServer() *server.Server {
	return ts.s
}
