# tRPC Middleware CL5

trpc-selector-cl5 是 trpc-selector的一种实现，提供trpc用户使用cl5进行路由以及负载均衡。需要安装 l5 agent。

[北极星](https://git.code.oa.com/polaris/polaris) 已经打通 l5 寻址，建议使用tRPC-Go 的[北极星插件](https://git.code.oa.com/trpc-go/trpc-naming-polaris)。

## example
```go
package main

import (
    _ "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    _ "git.code.oa.com/trpc-go/trpc-selector-cl5"

    pb "git.code.oa.com/trpcprotocol/test/helloworld"
)

func main() {
    ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
    defer cancel()

    opts := []client.Option{
        client.WithTarget("cl5://552833:917504"),
    }

    clientProxy := pb.NewGreeterClientProxy(opts...)

    req := &pb.HelloRequest{
        Msg: "client hello",
    }
    rsp, err := clientProxy.SayHello(ctx, req)
    log.Printf("req:%v, rsp:%v, err:%v", req, rsp, err)
}
```

