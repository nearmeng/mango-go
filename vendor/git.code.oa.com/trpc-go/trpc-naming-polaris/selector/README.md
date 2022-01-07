# 路由选择器插件

trpc-selector 的一种实现，提供trpc用户使用北极星进行路由以及负载均衡。
```go
package main

import (
	"context"
	"log"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/selector"

	pb "git.code.oa.com/trpcprotocol/test/helloworld"

	_ "git.code.oa.com/trpc-go/trpc-go"
)

func init() {
	selector.RegisterDefault()
}

func main() {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
	defer cancel()

	opts := []client.Option{
		client.WithNamespace("Development"),
		client.WithTarget("polaris://trpc.app.server.service"),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)

	req := &pb.HelloRequest{
		Msg: "client hello",
	}
	rsp, err := clientProxy.SayHello(ctx, req)
	log.Printf("req:%v, rsp:%v, err:%v", req, rsp, err)
}
```
