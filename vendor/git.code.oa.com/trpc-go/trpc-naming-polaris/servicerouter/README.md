# 服务路由

## 金丝雀功能

设计原理：https://git.code.oa.com/trpc/trpc-proposal/blob/master/A3-canary.md

- 配置开启
```
selector:                                          #针对trpc框架服务发现的配置
  polaris:                                         #北极星服务发现的配置
    enable_canary: true                           #开启金丝雀功能，默认 false 不开启
```

- 使用 demo
```go
package main

import (
    "context"
    "time"

    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/naming/registry"
    "git.code.oa.com/trpc-go/trpc-naming-polaris/servicerouter"

    pb "git.code.oa.com/trpcprotocol/test/helloworld"
)

func main() {
    ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond*2000)
    defer cancel()

    node := &registry.Node{}
    opts := []client.Option{
        client.WithServiceName("your service"),
        client.WithNamespace("Production"),
        client.WithSelectorNode(node),
        servicerouter.WithCanary("1"),
    }

    proxy := pb.NewGreeterClientProxy()

    req := &pb.HelloRequest{
        Msg: "trpc-go-client",
    }
    rsp, err := proxy.SayHello(ctx, req, opts...)
    log.Debugf("req:%s, rsp:%s, err:%v, node: %+v", req, rsp, err, node)
}
```

注意事项
- 1，确保使用框架的 ctx，否则金丝雀信息没办法传入下游
- 2，目前金丝雀仅在正式环境生效
- 3，有不理解的请先仔细阅读设计文档
- 4，问题定位，开启框架的 trace 日志，[开启方式请查看](https://git.code.oa.com/trpc-go/trpc-go/tree/master/log)，贴出[NAMING-POLARIS] 为前缀的日志。
