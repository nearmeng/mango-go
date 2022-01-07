## tRPC-Go [007](007.pcg.com)监控插件

[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-1b0813936a674b12a6bcb246bf226735/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](
http://api.devops.oa.com/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-1b0813936a674b12a6bcb246bf226735/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject) [![Coverage](
https://tcoverage.woa.com/api/getCoverage/getTotalImg/?pipeline_id=p-1b0813936a674b12a6bcb246bf226735)](http://macaron.oa.com/api/coverage/getTotalLink/?pipeline_id=p-1b0813936a674b12a6bcb246bf226735)

### m007监控平台介绍
* m007监控系统包含三个功能：
1. 模调上报(主调与被调)。类似于mic.oa.com这个老的模调系统
2. metrics属性上报。类似于monitor.oa.com
3. 自定义多维上报。
* 这三个功能从实现方式到使用方法都是不同的，请详细阅读以下说明

### 公共配置 trpc_go.yaml
* m007系统需要通过配置来初始化，以下配置是三个功能都需要的公共配置
```yaml
plugins:
  metrics:                                      #监控配置
    m007:                                       #007 monitor
      app:       test                           #业务名。[可选，未配置则与server.app一致]
      server:    helloworld                     #服务名。[可选，未配置则与server.server一致]
      ip:        127.0.0.1                      #本机IP。[可选，未配置则与global.local_ip一致]
      containerName:  container_name            #容器名称。[可选，未配置则与global.container_name一致]
      containerSetId:  set_id                   #容器SetId，支持多Set [可选，未配置则与full_set_name一致]
      version:   v0.0.1                         #应用版本 [可选，默认无]
      physicEnv: test-physicEnv                 #物理环境 [可选,未配置则与global.namespace一致]
      userEnv: test-userEnv                     #用户环境 [可选,未配置则与global.envname一致]
      frameCode: trpc                           #框架版本 trpc grpc等 [可选，默认为trpc]
      polarisAddrs: polaris-discover.oa.com:8090 #名字服务远程地址列表, ip1:port1,ip2:port2,ip3:port3[可选，默认为空，使用北极星默认埋点，IP由北极星SDK维护，不会变更]
      polarisProto: grpc #北极星交互协议支持 http，grpc，trpc[可选，默认为空，使用北极星默认埋点，IP由北极星SDK维护，不会变更]
```

* 最小配置如下，所有字段使用默认值
```yaml
plugins:
  metrics:                                      #监控配置
    m007:                                       #007 monitor
```

## 如何开发
### import插件包
* 不管是使用哪个功能，都必须在main文件import m007监控插件对应的package
    ```
        import _ "git.code.oa.com/trpc-go/trpc-metrics-m007"
    ```

### 模调上报
* trpc-go使用拦截器来实现了模调上报，使用方不需要写任何一行代码即可上报模调，只需要添加配置即可
1. 主调上报: 即client端的上报
```yaml
# client端添加007模调监控   
client:
  filter:
    - m007
```
2. 被调上报: 即server端的上报
```yaml
# server端添加007模调监控
server:
  filter:
    - m007
```
* 在007平台对应模块的【服务调用】tab即可看到上报的数据

### metrics属性上报
* metrics作为trpc-go的一个组件，实现为一个interface，m007的属性上报是其一个实现，对应的开源实现还有Prometheus等等
* 所以在使用的时候，先import m007，然后直接使用以下代码
    ```
        func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) (err error) {
            // 累积量
        	metrics.Counter("testCounter").Incr() 
            // 时刻量
            metrics.Gauge("testGauge").Set(1.2)
        ｝
    ```
* 这样的好处是：如果业务以后要换成其他的metrics系统，业务只需要改一下import即可，代码完全不用动  
* 在007平台对应模块的【特性指标】tab即可看到上报的数据
* 007SDK底层实现问题，所有metrics会统一上报全部策略（SET、AVG、MAX、MIN、SUM），为后面切换metrics系统方便，还是对应的metrics  
* 接口详见[trpc metrics](https://git.code.oa.com/trpc-go/trpc-go/blob/master/metrics/README.md)

### 自定义多维上报
* 007已适配框架多维上报，推荐使用框架接口，方便后续切换。代码示例
    ```
        var dimensions []*metrics.Dimension
        dimensions = append(dimensions, &metrics.Dimension{Name: "cmd", Value: "concreteCmd"})    // 命令字
        dimensions = append(dimensions, &metrics.Dimension{Name: "dimension2", Value: "concreteDimension2"}) // 维度2
        var values []*metrics.Metrics
        values = append(values, metrics.NewMetrics("totalCount", 1, metrics.PolicySUM)) // 总量，可查看具体维度（命令字）值对应的请求量
        rec := metrics.NewMultiDimensionMetricsX("request", dimensions, values)
        _ = metrics.Report(rec)
    ```
* 007SDK底层实现问题，dimensions, values对应的Name不会上报，需要去007管理端配置（默认名称类似d7, v0）
* 007SDK会默认补充7个维度，见[2.4【框架】自定义上报汇聚格式](https://iwiki.woa.com/pages/viewpage.action?pageId=98470173),服务名、IP等trpc_go.yaml配置属性
* metrics同样只支持（SET、AVG、MAX、MIN、SUM）五种策略
* 在007平台对应模块的【特性指标】->自定义 tab即可看到上报的数据

### 自定义错误码处理函数

覆盖 m007.DefaultGetStatusAndRetCodeFunc 可自定义007监控使用的status和code的计算方式, 适合业务handler没有返回error但需要使用007主调被调监控的场景
示例:

```go
func defaultCodeStatus(code int32) (m007.Status, string) {
	if code == 0 {
		return m007.StatusSuccess, strconv.Itoa(int(code))
	}
	return m007.StatusException, strconv.Itoa(int(code))
}

func init() {
	m007.DefaultGetStatusAndRetCodeFunc = func(ctx context.Context,
		req interface{}, rsp interface{}, err error) (m007.Status, string) {

		if err != nil {
			return m007.GetStatusAndRetCodeFromError(err)
		}
		switch v := rsp.(type) {
		case interface {
			GetRetcode() int32
		}:
			return defaultCodeStatus(v.GetRetcode())
		case interface {
			GetRetCode() int32
		}:
			return defaultCodeStatus(v.GetRetCode())
		case interface {
			GetCode() int32
		}:
			return defaultCodeStatus(v.GetCode())
		default:
			return defaultCodeStatus(0)
		}
	}
}
```

### 上报自定义维度到模调监控

使用m007.SetExtensionDimension将需要上报的维度值信息设置到主被调接口的context中，即可将自定义维度的内容上报到主被调监控的预留三字段中。

代码示例：
```go 
import (
    m007 "git.code.oa.com/trpc-go/trpc-metrics-m007"
)

func xxx(ctx context.Context) {
    m007.SetExtensionDimension(ctx, "metrics_test")
}
```

### 注意
* 数据通过atta上报，使用前请确保atta agent已安装，详见[如何安装AttaAgent](http://km.oa.com/articles/show/447456?kmref=search&from_page=1&no=3)
* 若007服务调用，特性指标没有自动出现监控项，请到监控搜索那里输入 关键词( {app}.{server} ) 搜索获取对应链接
* 123平台默认已经安装了attaagent

### 查看007监控
* http://007.pcg.com/#/
