# 负载均衡插件

一致性hash 或者普通hash 负载均衡方式使用如下：
```go
import (
	_ "git.code.oa.com/trpc-go/trpc-naming-polaris"
)

func main() {
	opts := []client.Option{
		// 命名空间
		client.WithNamespace("Development"),
		// 服务名
		client.WithServiceName("trpc.app.server.service"),
		// 普通 hash
		// client.WithBalancerName("polaris_hash"),
		// 一致性hash，支持枚举请参考 
		// https://git.code.oa.com/trpc-go/trpc-naming-polaris/blob/master/loadbalance/loadbalance.go#L19
		client.WithBalancerName("polaris_ring_hash"),
		// hash key 
		client.WithKey("your hash key"),
	}

	clientProxy := pb.NewGreeterClientProxy(opts...)
	req := &pb.HelloRequest{
		Msg: "hello",
	}

	rsp, err := clientProxy.SayHello(ctx, req)
	if err != nil {
		log.Error(err.Error())
		return 
	}

	log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
}
```

!`一致性 hash 不生效`，请先升级到最新版本插件。
