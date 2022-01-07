# Change Log

## [0.4.7](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.4.7) (2021-12-09)

### Features
- 处理业务本身返回的错误就是取消或超时的情况
- 优化注释
- 解决单测无断言问题

## [0.4.6](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.4.6) (2021-10-15)

### Features
- 区分客户端超时和实际服务超时错误码

## [0.4.5](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.4.5) (2021-08-16)

### Features 
- 上报target/address字段到007
- 007添加日志插件
- 支持最新版本的attaapi

## [0.4.4](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.4.4) (2021-05-14)

### Features 
- 更新007SDK，拉取远程配置超时时间调整为2s，删除失败对应本地日志

## [0.4.3](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.4.3) (2021-05-14)

### Bug Fixes 
- 修复time.Duration<1ms时的上报为0的问题

## [0.4.2](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.4.2) (2021-04-16)

### Features 
- 被调支持超时上报
- 主调上报新增被调Set信息

## [0.4.1](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.4.1) (2020-12-31)

### Features 
- 北极星配置空 默认使用埋点，减少域名依赖

## [0.4.0](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.4.0) (2020-12-24)

### Bug Fixes 
- 北极星SDK BUG fix

## [0.3.2](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.3.2) (2020-12-24)

### Features
- 外部依赖导致007初始化失败会注册空filter,不阻塞整体流程
- SDK版本升级到v0.3.2：中控配置细化到监控项粒度、删除无意义日志

## [0.2.4](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.2.4) (2020-09-13)

### Features
- 007SDK升级到v0.2.4：atta api初始化返回值细化、数据降维远程配置格式调整
- 支持自定义错误码处理函数

## [0.2.3](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.2.3) (2020-07-21)

### Features
- 007SDK升级，支持远程日志，debuglog开关等

## [0.2.2](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.2.2) (2020-04-25)

### Features
- 支持SetName上报

## [0.2.1](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.2.1) (2020-04-11)

### Features
- 007SDK升级，单维度值数量限制

# Change Log

## [0.2.0](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.2.0) (2020-04-04)

### Features
- 007支持框架多维度上报
- 北极星接入修改为域名方式

## [0.1.6](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.1.6) (2020-03-20)

### Features
- 更新007 report api版本
- 属性上报兼容新接口

## [0.1.5](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.1.5) (2020-03-18)

### Bug Fixes
- 解决编译问题，更新trpc-go到v0.2.0

## [0.1.4](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.1.4) (2020-03-02)

### Features
- 添加插件统计数据上报

## [0.1.3](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.1.3) (2020-02-13)

### Features
- 添加错误码前缀，方便区分不同类型的错误码

## [0.1.2](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.1.2) (2020-02-06)

### Bug Fixes 
- 上报超时状态时，没有区分框架错误码和业务错误码，导致误报

## [0.1.1](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.1.1) (2020-01-17)

### Features
- 框架错误码上报007加上trpc_前缀，以区分开业务错误码上报

## [0.1.0](https://git.code.oa.com/trpc-go/trpc-metrics-m007/tree/v0.1.0) (2020-01-13)

### Features
- 支持模调上报（主调、被调）
- 支持属性全策略上报
- 支持自定义多维上报
