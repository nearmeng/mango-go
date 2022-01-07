package module

import (
	"context"

	"github.com/nearmeng/mango-go/example/statelesssvr/proto/echo"
	"github.com/nearmeng/mango-go/plugin/log"
)

type echoServiceImpl struct{}

func (s *echoServiceImpl) Echo(ctx context.Context, req *echo.EchoReq, rsp *echo.EchoRsp) error {
	// implement business logic here ...
	// ...

	log.Info("req is %s\n", req.GetMessage())
	rsp.Response = "hi " + req.Message

	return nil
}
