## [0.3.13](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.13) (2021-07-30)
### Features
- 上报日志方式改为队列上报

## [0.3.12](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.12) (2021-07-22)
### Features
- 解决初始化首次获取中控信息失败问题

## [0.3.11](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.11) (2021-07-15)
### Features
- 日志上报按日志名称上报，去除日志级别纬度
- 封装日志对外提供的接口

## [0.3.10](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.9) (2021-06-21)
### Features
- 框架上报增加自定义维度 字段

## [0.3.9](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.9) (2021-06-21)
### Features
- atta库升级attaapi-go

## [0.3.6](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.5) (2021-05-30)
### Features
- 新增日志上报

## [0.3.5](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.5) (2021-05-14)

### Fix
- 远程日志 超时时间调整为2s，删除相关本地日志


## [0.3.4](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.4) (2020-12-29)

### Features
- 更新北极星，支持默认埋点，减少域名依赖

### Fix
- 批量发送按attaID汇聚，减少发送次数
- 内部atta log截断


## [0.3.3](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.3) (2020-12-24)

### Features
- 中控细化到监控项粒度
- 重构，支持单进程 多上报实例
- 删除无意义本地log

## [0.3.1](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.3.1) (2020-10-13)

### Features
- 采样

## [0.2.4](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.2.3) (2020-09-13)

### Features
- attaapi初始化返回值细分
- 中控配置 数据降维调整
- 业务数据上报接口

## [0.2.3](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.2.3) (2020-07-21)

### Features
- 支持DebugLog配置

## [0.2.2](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.2.2) (2020-07-01)

### Features
- 远程日志上报
- 模调支持耗时区间
- 管道合并

### Fix
- 属性&自定义上报 数据降维

## [0.2.1](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.2.1) (2020-06-24)

### Fix
- 修复atta初始化卡死的bug

## [0.2.0](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.2.0) (2020-04-03)

### Features
- 增加对单维度值的数量限制

## [0.1.2](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.1.2) (2020-04-03)

### Features
- 北极星配置默认值调整为域名方式
- StatValue Count字段强制为1

## [0.1.1](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.1.1) (2020-03-19)

### Features
- [非框架]上报,支持多监控项
- [框架]属性上报接口修改，不在区分策略，007全策略上报
- [框架]模调上报接口修改
- [非框架]自监控上报

## [0.1.0](https://git.code.oa.com/pcgmonitor/trpc_report_api_go/tree/v0.1.0) (2020-01-13)

### Features
- 支持[框架]模调上报（主调、被调）
- 支持[框架]属性全策略上报
- 支持[框架]自定义多维上报
- 支持[非框架]上报