# tRPC-Go 北极星名字服务插件
[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-8dcb44954edd4414923421fafce08e48/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com:/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-8dcb44954edd4414923421fafce08e48/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject) [![Coverage](https://tcoverage.woa.com/api/getCoverage/getTotalImg/?pipeline_id=p-8dcb44954edd4414923421fafce08e48)](http://macaron.oa.com/api/coverage/getTotalLink/?pipeline_id=p-8dcb44954edd4414923421fafce08e48) [![GoDoc](https://img.shields.io/badge/API%20Docs-GoDoc-green)](http://godoc.oa.com/git.code.oa.com/trpc-go/trpc-naming-polaris)

包括了“服务注册、服务发现、负载均衡、熔断器”等组件，通过框架配置可以在 tRPC-Go 框架内部使用，也可以整体使用。

文档也可以查看 https://iwiki.woa.com/pages/viewpage.action?pageId=284289117 第五章。

北极星已经打通 l5、ons 寻址，建议使用北极星插件寻址。
l5 寻址插件也可以使用：[trpc-selector-cl5](https://git.code.oa.com/trpc-go/trpc-selector-cl5)
ons 寻址插件也可以使用：[trpc-selector-ons](https://git.code.oa.com/trpc-go/trpc-selector-ons)
cmlb 寻址插件也可以使用：[trpc-selector-cmlb](https://git.code.oa.com/trpc-go/trpc-selector-cmlb)

## 123平台部署默认`不需要做任何配置`，只需引入即可。
引入北极星插件

```go
import (
	_ "git.code.oa.com/trpc-go/trpc-naming-polaris"
)
```
如果需要在其他平台（例如织云）使用请参考最后的服务完整配置实例。

### !`注意`， `l5, ons` 的 namespace 为 `Production`，且必须关闭服务路由，如下:

```go
opts := []client.Option{
    client.WithNamespace("Production"),
    // trpc-go 框架内部使用
    client.WithServiceName("12587:65539"),
    // 纯客户端或者其他框架中使用trpc-go框架的 client
    // client.WithTarget("polaris://12587:65539"),
    client.WithDisableServiceRouter(),
}
```

## 服务寻址
### `tRPC-Go 框架内（tRPC-Go 服务）`寻址
```go
import (
	_ "git.code.oa.com/trpc-go/trpc-naming-polaris"
)

func main() {
	opts := []client.Option{
		// 命名空间，不填写默认使用本服务所在环境 namespace
		// l5, ons namespace 为 Production
		client.WithNamespace("Development"),
		// 服务名
		// l5 为 sid
		// ons 为 ons name
		client.WithServiceName("trpc.app.server.service"),
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

### 获取被调 ip
```go
import (
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"

	_ "git.code.oa.com/trpc-go/trpc-naming-polaris"
)

func main() {
	node := &registry.Node{}
	opts := []client.Option{
		client.WithNamespace("Development"),
		client.WithServiceName("trpc.app.server.service"),
		// 传入被调 node
		client.WithSelectorNode(node),
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
	// 打印被调节点
	log.Infof("remote server ip: %s", node)

	log.Info("req:%v, rsp:%v, err:%v", req, rsp, err)
}
```

### 服务注册

https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/master/registry

### 负载均衡

一致性hash 或者普通hash 负载均衡方式使用如下：

https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/master/loadbalance

### 熔断器

https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/master/circuitbreaker

### 熔断探活
节点进入熔断后可通过以下两种途径进入半开状态（业务探测状态）
1. 通过超时进入半开：用户没有开启主动探测的前提下，熔断的节点在时间窗（sleepWindow）过了后，会自动进入半开状态
2. 通过主动探测进入半开：假如用户开启了主动探测（默认不开启），则内部会启动定时任务，对熔断的节点进行探测，探测通过后，节点进入半开状态；

熔断探活就是主动探测机制，北极星插件提供了基于TCP连接的探测机制，配置如下：
```
selector:                                          #针对trpc框架服务发现的配置
  polaris:                                         #北极星服务发现的配置
    outlierDetection:
      enable: true                                 #是否开启熔断探活功能
      checkPeriod: 12s                             #定时熔断探活探测周期
```
关于熔断和熔断探活机制的详细描述，请参考[北极星iwiki](https://iwiki.woa.com/pages/viewpage.action?pageId=89656470)的描述

## 指定环境请求
在关闭服务路由的前提下，可以通过设置环境名来指定请求路由到具体某个环境。
关于如何关闭服务路由可查看[多环境路由](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=99485673) 。
```
opts := []client.Option{
	// 命名空间，不填写默认使用本服务所在环境 namespace
	// l5, ons namespace 为 Production
	client.WithNamespace("Development"),
	// 服务名
	// l5 为 sid
	// ons 为 ons name
	// client.WithTarget("polaris://trpc.app.server.service"),
	client.WithServiceName("trpc.app.server.service"),
	// 设置被调服务环境
	client.WithCalleeEnvName("62a30eec"),
	// 关闭服务路由
	client.WithDisableServiceRouter()
}
```

## 多环境路由 

https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=99485673

### 在`其他框架或者服务`使用进行寻址
https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/master/selector

### 配置完整示例。
`registry` 为服务注册相关的配置，`selector`为服务寻址相关的配置。

```yaml
plugins:                                             #插件配置
  registry:
    polaris:                                         #北极星名字注册服务的配置
      heartbeat_interval: 3000                       #名字注册服务心跳上报间隔
      protocol: grpc                                 #名字服务远程交互协议类型
      #connect_timeout: 1000                         #单位 ms，默认 1000ms，连接北极星后台服务的超时时间
      #message_timeout: 1s                           #类型为 time.Duration，从北极星后台接收一个服务信息的超时时间，默认为1s
      #join_point: default                           #名字服务使用的接入点,该选项会覆盖 address_list 和 cluster_service
      
  selector:                                          #针对trpc框架服务发现的配置
    polaris:                                         #北极星服务发现的配置
      #debug: true                                   #开启 debug 日志
      #enable_canary: false                          #开启金丝雀功能，默认 false 不开启
      #timeout: 1000                                 #单位 ms，默认 1000ms，北极星获取实例接口的超时时间
      #report_timeout: 1ms                           #默认1ms，如果设置了，则下游超时，并且少于设置的值，则忽略错误不上报
      #connect_timeout: 1000                         #单位 ms，默认 1000ms，连接北极星后台服务的超时时间
      #message_timeout: 1s                           #类型为 time.Duration，从北极星后台接收一个服务信息的超时时间，默认为1s
      #log_dir: $HOME/polaris/log                    #北极星日志目录
      protocol: grpc                                 #名字服务远程交互协议类型
      #join_point: default                           #接入名字服务使用的接入点，该选项会覆盖 address_list 和 cluster_service
      #enable_servicerouter: true                    #是否开启服务路由，默认开启
      #persistDir: $HOME/polaris/backup              #服务缓存持久化目录，按照服务维度将数据持久化到磁盘
      #service_expire_time: 24h                      #服务缓存的过期淘汰时间，类型为 time.Duration，如果不访问某个服务的时间超过这个时间，就会清除相关服务的缓存
      #loadbalance: 
      #  name:                                       #负载均衡类型，默认值
      #    - polaris_wr                              #加权随机，如果默认设置为寻址方式，则数组的第一个则为默认的负载均衡
      #    - polaris_hash                            #hash算法
      #    - polaris_ring_hash                       #一致性hash算法
      #discovery:
      #  refresh_interval: 10000                     #刷新间隔，毫秒
      #cluster_service:
      #  discover: polaris.discover                  #修改发现server集群名
      #  health_check: polaris.healthcheck           #修改心跳server集群名
      #  monitor: polaris.monitor                    #修改监控server集群名
      #circuitbreaker:
      #  checkPeriod: 30s                             #实例定时熔断检测周期, 默认值:30s
      #  requestCountAfterHalfOpen: 10                #熔断器半开后最大允许的请求数, 默认值:10
      #  sleepWindow: 30s                             #熔断器打开后，多久后转换为半开状态，默认值:30s
      #  successCountAfterHalfOpen: 8                 #熔断器半开到关闭所必须的最少成功请求数，默认值:8
      #  chain:                                       #熔断策略，默认值：[errorCount, errorRate]
      #    - errorCount                               #基于周期连续错误数熔断
      #    - errorRate                                #基于周期错误率的熔断
      #  errorCount:
      #    continuousErrorThreshold: 10               #触发连续错误熔断的阈值，默认值:10
      #    metricNumBuckets: 10                       #连续错误数的最小统计单元数量，默认值:10
      #    metricStatTimeWindow: 1m0s                 #连续失败的统计周期，默认值:1m
      #  errorRate:
      #    metricNumBuckets: 5                        #错误率熔断的最小统计单元数量，默认值:5
      #    metricStatTimeWindow: 1m0s                 #错误率熔断的统计周期，默认值:1m
      #    requestVolumeThreshold: 10                 #触发错误率熔断的最低请求阈值，默认值:10
      #outlierDetection:
      #  enable: true                                 #是否开启熔断探活功能
      #  checkPeriod: 12s                             #定时熔断探活探测周期
      #service_router:
      #  nearby_matchlevel: zone                      #就近路由的最小匹配级别, 包括region(大区)、zone(区域)、campus(园区), 默认为zone
      ## WithTarget模式下，trpc协议透传字段传递给北极星用于meta匹配的开关
      ## 开启设置，则将'selector-meta-'前缀的透传字段摘除前缀后，填入SourceService的MetaData，用于北极星规则匹配
      ## 示例：透传字段 selector-meta-key1:val1 则传递给北极星的meta信息为 key1:val1
      #enable_trans_meta: true                        
```

### l5 ons 已经打通北极星
- l5
    - namespace: Production
    - serviceName: sid
- ons
    - namespace: Production
    - serviceName: ons name
- cmlb 
	- namespace: Production
	- serviceName: cmlb id 

### 北极星服务发现详细文档
https://git.code.oa.com/polaris/polaris/wikis/home

### 北极星相关插件mock命令
- mockgen git.code.oa.com/polaris/polaris-go/api SDKContext,ConsumerAPI,ProviderAPI > mock/mock_api/api_mock.go
- mockgen git.code.oa.com/polaris/polaris-go/pkg/plugin/loadbalancer LoadBalancer > mock/mock_loadbalancer/loadbalancer_mock.go
- mockgen git.code.oa.com/polaris/polaris-go/pkg/model Instance,CircuitBreakerStatus,ServiceInstances,ValueContext,ServiceClusters > mock/mock_model/model_mock.go
- mockgen git.code.oa.com/polaris/polaris-go/pkg/plugin Manager,Plugin > mock/mock_plugin/plugin_mock.go
- mockgen git.code.oa.com/polaris/polaris-go/pkg/plugin/servicerouter ServiceRouter > mock/mock_servicerouter/servicerouter_mock.go 
