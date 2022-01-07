# 天机阁 Go SDK [![GoDoc](https://pkg.woa.com/badge/git.code.oa.com/tpstelemetry/tps-sdk-go)](https://pkg.woa.com/git.code.oa.com/tpstelemetry/tps-sdk-go)

天机阁(TpsTelemetry)，分布式全链路遥测云原生系统。
- 全链路：客户端，前端，接入层，后台逻辑，数据存储
- 遥测(telemetry)：涵盖trace跟踪数据，log日志数据，metrics指标数据

## 快速上手

### 1. 使用trpctelemetry方式接入 (tRPC框架推荐使用此种方式)

1. 匿名引入天机阁tRPC拦截器

```go
import _ "git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry"
```

2. 修改tRPC框架配置文件

```yaml
server:
  filter:
    - tpstelemetry         #在tRPC服务端处理过程，引入天机阁拦截器

client:
  filter:
    - tpstelemetry         #在tRPC客户端处理过程，引入天机阁拦截器

plugins:
  log:                                            #日志配置
    default:                                      #默认日志的配置，可支持多输出
      - writer: file                              #本地文件日志
        level: error                               #本地文件滚动日志的级别
        writer_config:
          filename: ../log/trpc.log                 #本地文件滚动日志存放的路径
          max_size: 10                              #本地文件滚动日志的大小 单位 MB
          max_backups: 10                           #最大日志文件数
          max_age: 7                                #最大日志保留天数
          compress:  false                          #日志文件是否压缩
      - writer: tpstelemetry                        # 远程日志
        level: error                                # 优先选择本日志级别，未配置会使用telemetry中的level

  telemetry: # 注意缩进层级关系
    tpstelemetry:
      addr: otlp.tpstelemetry.woa.com:12520  # 天机阁集群地址（检查环境域名是否可以正常解析）
      tenant_id: your-tenant-id              # 租户ID，default代表默认租户，（注意：切换为业务租户ID）
      sampler:
        fraction: 0.0001                     # 万分之一采样（每10000请求上报一次trace数据）
        sampler_server_addr: apigw.tpstelemetry.woa.com:14941    # 天机阁染色元数据查询平台地址
      metrics:
        # 天机阁metrics注册地址 metrics功能需要打开trpc_admin, 如果运行在123平台, 则自动开启
        registry_endpoints: ["registry.tpstelemetry.woa.com:2379"]
        server_owner: # 服务负责人, 对于123平台会自动设置. 用于监控看板展示及告警. 多个以分号分隔
        # sdk 0.3.5 版本开始 code_type_mapping 字段标记为废弃, 应使用codes, 旧配置会兼容处理
        # codes 可设置特定错误码的类型(错误码转义), 以便计算错误率/超时率/成功率和看板展示错误码描述
        # 默认值: 0:成功success 21/101:超时timeout 其它:错误exception
        codes:
          - code: 21
            type: timeout
            description: server超时
          - code: 101
            type: timeout
            description: client超时
        # 下面为设置特定返回码的例子，业务可按需设置
          - code: 100014
            # type 为 success 表示 100014 这个返码(无论主被调)会被统计为成功
            # 不区分主被调，如果担忧错误码冲突，可以设置 service 和 method 来限定生效的 service 和 method
            type: success 
            description: desc4 # 对这个返回码的具体描述
          - code: 100015
            type: exception # type 为 exception 表示 100015 是个异常的错误码。可在description里设置更详细的说明信息
            description: desc5
            service: # 不为空表示错误码特例仅匹配特定的 service, 为空表示所有 service
            method: # 不为空表示错误码特例仅匹配特定的 method, 为空表示所有 method
      logs:
        enabled: true # 天机阁远程日志开关，默认关闭
        level: "info" # 天机阁日志级别，默认error
        enable_sampler: false # 是否启用采样器, 启用后只有当前请求命中采样时才会上报独立日志
        # 天机阁trace_log(follow log)模式,  枚举值可选:verbose/disable
        # verbose:以DEBUG级别打印flow log包括接口名、请求、响应、耗时, disable:不打印, 默认不打印
        trace_log_mode: "verbose"
        disable_recovery: false # log filter默认会recovery panic并打印日志上报指标
        # rate_limit 为日志流控配置，开启此配置可减少重复日志的打印
        # 例如，tick = 1s，first = 10, thereafter = 10 表示1秒内同一条日志打印超过10条后，则每隔10条才再次打印这一条相同的日志
        # 此时在这1s的生效周期里，如果某个相同的日志本应打印100条，实际上传的条数为19
        # 这里定义"相同的日志"为内容和等级都完全相同的重复日志。注意这里不包括日志的fields。如果fields不同，但内容和等级相同也被视为"相同的日志"
        rate_limit:
          enable_rate_limit: false # 是否开启日志流控配置。若开启，请按照业务需求来配置tick，first和thereafter
          tick: 1s # tick是日志流控的生效周期（即从打印一条日志开始计时在tick时间后，无论触发限流与否，对同一条计数器会被置为零，重新开始计数)
          first: 100 # first是限流阈值，即相同的日志达到first条时触发限流
          thereafter: 3 # 触发限流后每thereafter条相同日志才会输出一条
      traces:
        disable_trace_body: false # 天机阁trace对req和rsp的上报开关, true:关闭上报以提升性能, false:上报, 默认上报
        enable_deferred_sample: false # 是否开启延迟采样 在span结束后的导出采样, 额外上报出错的/高耗时的. 默认: disable
        deferred_sample_error: true # 采样出错的
        deferred_sample_slow_duration: 500ms # 采样耗时大于指定值的
```

### 2. 使用 tpstelemetry sdk方式接入

如果业务使用的框架没有实现类似 trpc-go 的上报插件，也可直接使用 tpstelemetry sdk 方式接入。 上报demo可参考 [example](./example)。