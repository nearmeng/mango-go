## [0.4.16](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.16) (2021-12-10)
- feat: 序列化失败时,不再尝试使用%s打印请求包体,避免req持有ctx导致竞争问题

## [0.4.14](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.14) (2021-11-03)

### Features
- feat: client span上报的trpc主被调method只信任RegisterMethodMapping的pattern，避免大量脏数据写入拓扑存储中

### Features
- feat: client span上报的trpc主被调method只信任RegisterMethodMapping的pattern，避免大量脏数据写入拓扑存储中

## [0.4.13](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.13) (2021-11-02)

### Features
- fix: 重新打tag避免v0.4.12版本的check sum冲突问题，请不要使用v0.4.12版本

## [0.4.12](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.12) (2021-11-01)

### Features
- feat: 天机阁远程日志支持流控
- feat: 修改trpc-go插件部署在123平台上服务的owner逻辑，只考虑服务负责人和运维负责人

## [0.4.11](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.11) (2021-10-28)

### Features
- fix: 修改重试间隔计算
- feat: span上报的trpc主被调method只信任RegisterMethodMapping的pattern，避免大量脏数据写入拓扑存储中

## [0.4.10](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.10) (2021-10-14)

### Features
- feat: 调整moduleID最新labels标识
- feat: 优化trpc metrics api映射至Prometheus的模型 (merge request !194)
- feat: 完善owner合法性校验

### Changes
- trpc metrics counter映射Prometheus指标修改为 `trpc_counter_total{_name="原始的中文指标名"}` , 
  如果用户的查询语句使用旧的拼音指标名, 需要增加or子句兼容.
  ```promql
  sum(increase(trpc_counter_total{_name="原始的中文指标名",_type="counter",app="app",server="server"}[1m])) by (_name)
  or # 以下兼容旧指标
  sum(increase({__name__!="trpc_counter_total", _name="原始的中文指标名",_type="counter",app="app",server="server"}[1m])) by (_name)
  ```
- trpc metrics gauge映射Prometheus指标修改为 `trpc_gauge{_name="原始的中文指标名"}`,
  如果用户的查询语句使用旧的拼音指标名, 需要同步修改, 如使用`{_name="原始的中文指标名"}`则自动兼容无需修改.
- trpc metrics timer/histogram映射的Prometheus指标名增加了 `trpc_histogram_` 前缀, 需要增加or子句兼容.

## [0.4.9](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.9) (2021-09-08)

### Features
- fix: Gometalinter 修改
- feat: 为logMode实现MarshalText方法
- feat: 修改log的batch processor逻辑，修复若干问题
- feat: 支持stke通过环境变量或labels上报模块ID

## [0.4.8](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.8) (2021-08-26)

### Features

- fix: 更改go.mod的stdout版本

## [0.4.7](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.7) (2021-08-26)

### Features

- feat: 更新demo文档
- feat: kafka增加baggage能力
- fix: 修复在非容器环境下获取内存数据时可能panic的问题
- fix: 支持不同log的tpstelemetry writer设置不同的日志级别 --bug=56 closes issue #56 (merge request !173)
- feat: 染色上报请求应答包,无需限制为proto.Message (merge request !175)
- feat: register sdk支持完全自定义key (merge request !171)

## [0.4.6](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.6) (2021-08-06)

### Features
- feat: 日志otlp导出时附加x-tps-tenantid header.
- feat: 支持染色日志(日志采样)
- feat: log支持Shutdown

## [0.4.5](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.5) (2021-07-26)

### Features
- feat: 实现延迟采样, 单实例的tail-based-sampling.

### Bug Fixes
- fix: 修复trpc client span未取到peer ip作为attribute的问题.

## [0.4.4](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.4) (2021-07-21)

### Features
- feat: 兼容123平台的查询服务详情接口的特性，防止服务负责人里出现"NONE"
- feat: etcd注册时带上header x-tps-tenantid
- feat: 兼容 [tenc](http://scr.ied.com/#/) 平台，使之能够正确获取内存分配量

## [0.4.3](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.3) (2021-07-19)

### Features
- feat: trpc-go在v0.7.0之后的版本会废弃admin.Run。对其进行兼容，废弃trpctelemetry中的admin.Run

## [0.4.2](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.2) (2021-07-15)

### Features
- feat: 增加sdk版本、请求包/metadata大小监控
- feat: 增加gin instrumentation
- feat: traces/log 根据执行结果是否 OK 调整打印日志级别

## [0.4.1](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.1) (2021-07-07)

### Features

### Bug Fixes
- 修复metrics支持导出exemplar的codeType条件判断错误.

## [0.4.0](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.4.0) (2021-07-06)

### Features
- feat: 非容器环境 memory quota 监控
- feat: 注册Etcd支持设置后缀，用于支持同一地址多个被抓取源
- feat: span processor 支持按照流量阈值导出

### Bug Fixes
- global tracer默认值


## [0.3.8](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.8) (2021-06-17)

### Features
- metrics: 支持导出exemplar.
- trpc instrumentation & taf instrumentation: 支持对 rpc code!=success 的请求在 rpc_*_handled 系列指标中自动增加exemplar. 

### Bug Fixes


## [0.3.7](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.7) (2021-06-16)

### Features
- trpc instrumentation: 天机阁recovery行为开关可配置, 支持自定义RecoveryHandler函数.
- trpc instrumentation: trpc metric为单维metric, 构造时可直接柯里化, 提升性能
- metrics: 默认基数限制特殊场景可修改.

### Bug Fixes


## [0.3.6](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.6) (2021-06-07)

### Features

### Bug Fixes
- otelgrpc exporter可能会发生gogoprotobuf序列化引起的panic, 对单次export请求加个recovery打印日志定位问题.


## [0.3.5](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.5) (2021-06-07)

### Features
- metrics: 错误码描述信息支持从远端配置, 为支持gitops奠定基础.
- metrics: 重构错误码描述信息配置, trpc instrumentation code_type_mapping 标记为废弃, 使用新的 codes 列表结构.

### Bug Fixes
- 修复设置spanStatus未按枚举定义设置的问题.
- 修复v0.2.6支持kafka透传trace上下文时引入的染色逻辑异常问题.


## [0.3.4](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.4) (2021-05-24)

### Features

### Bug Fixes
- trpc instrumentation: 修复trpc Metrics API导出为Prometheus指标时重复创建对象的问题.


## [0.3.3](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.3) (2021-05-24)

### Features
- 将引入的git.woa.com包移动到git.code.oa.com, 方便用户编译

### Bug Fixes
- 修复 0.3.2 go.mod 未更新的问题


## [0.3.2](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.2) (2021-05-22)

### Features
- 将引入的git.woa.com包移动到git.code.oa.com, 方便用户编译

### Bug Fixes


## [0.3.1](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.1) (2021-05-21)

### Features

### Bug Fixes
- trpc instrumentation: 修复trpc Metrics API导出为Prometheus指标时误添加了log的问题


## [0.3.0](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.3.0) (2021-05-21)

### Features
- trpc instrumentation: 支持trpc Metrics API导出为Prometheus指标.
- metrics: 添加收集容器内存总量.

### Bug Fixes


## [0.2.7](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.2.7) (2021-05-17)

### Features
- metrics sdk Prometheus导出器支持基数限制.

### Bug Fixes


## [0.2.6](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.2.6) (2021-05-17)

### Features
- trpc instrumentation: 添加jsoniter extension使得默认的req rsp的序列化函数支持将int64序列化为string, 解决int64大数在前端展示问题.
- trpc instrumentation: server_owner支持列表.
- trpc instrumentation: 优化filter性能, 非采样且不打印流水日志时不序列化req rsp.
- trpc instrumentation: 增加 traces.disable_trace_body 选项关闭序列化及上报req rsp.
- metrics sdk Prometheus导出器支持基数限制.
- 增加metadata metrics放置版本信息.

### Bug Fixes
- 修复 otel 0.19.0 SDK默认的batchProcessor内存泄露问题.


## [0.2.5](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.2.4) (2021-04-22)

### Features
- 允许自定义req rsp的序列化函数

### Bug Fixes


## [0.2.4](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.2.4) (2021-04-19)

### Features
- trpc instrumentation支持错误码描述信息

### Bug Fixes


## [0.2.2](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.2.2) (2021-04-08)

### Features
- 升级OpenTelemetry SDK 至 0.19.0
- 添加going instrumentation支持
- 调整grpc请求体上限

### Bug Fixes
- 解决日志sampled属性上报错误


## [0.2.1](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.2.1) (2021-03-17)

### Features
- 升级OpenTelemetry SDK 至 0.18.0

### Bug Fixes
- 解决日志sampled属性上报错误

## [0.2.0](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.2.0) (2021-01-30)

### Features
- 升级OpenTelemetry SDK 至 0.16.0

### Bug Fixes
- 解决日志sampled属性上报错误

## [0.1.0](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.1.0) (2021-01-07)

### Features
- 升级OpenTelemetry SDK 至 0.15.0
- 添加taf instrumentation支持

### Bug Fixes

## [beta](https://git.code.oa.com/tpstelemetry/tps-sdk-go/tree/v0.1.0) (2021-01-07之前)
