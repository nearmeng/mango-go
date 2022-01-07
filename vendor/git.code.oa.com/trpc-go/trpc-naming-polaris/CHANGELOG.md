# Change Log
## [0.3.1](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.3.1) (2021-12-02)
### features
- 北极星插件 withservicename 模式增加 plugin
- 支持注册自定义selector & 新增LocalCachePersistDir选项

## [0.3.0](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.3.0) (2021-09-02)
### features
- 多环境路由，新增透传 trpc 协议字段到北极星 meta 的能力

## [0.2.13](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.13) (2021-08-24)
### bugs
- 修复环境透传只透传优先级最高环境的问题

## [0.2.12](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.12) (2021-07-21)
### features
- tRPC-Go 全链路超时区分全链路和用户定义的超时，减少错误服务熔断判定

## [0.2.11](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.11) (2021-07-02)
### features
- 支持配置开启金丝雀功能

## [0.2.10](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.10) (2021-06-21)
### features
- 添加对于北极星缓存超时和接入点配置的支持
- 北极星熔断探活功能
- 北极星插件增加就近路由的最小匹配级别参数

## [0.2.9](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.9) (2021-05-25)
### features
- 支持北极星连接超时
- 提供外部可访问的新建实例接口
- 修复测试失败
- 优化文档

## [0.2.8](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.8) (2021-04-07)
### features
- selector 暴露 UseBuildin 

## [0.2.7](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.7) (2021-03-26)
### features
- registry 支持指定区域

## [0.2.6](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.6) (2021-03-26)
### features
- 修复指定 env 不生效问题

## [0.2.5](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.5) (2021-03-17)
### Features
- 升级sdk版本到 0.8.1

## [0.2.4](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.4) (2021-03-12)
### Features
- 升级sdk版本到 0.8.0
- 负载均衡支持虚拟节点数
- 支持金丝雀功能

## [0.2.2](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.2) (2020-11-27)
### Features
- 升级sdk版本
- 增加 l5cst, maglv hash

## [0.2.1](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.1) (2020-08-05)
### Features
- 升级sdk版本
- 不依赖外部传入的地址

## [0.2.0](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.2.0) (2020-08-05)
### Features
- 升级框架版本
- 默认不注册selector

## [0.1.16](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.16) (2020-08-05)
### Features
- 升级框架版本
- 路由规则支持metadata

## [0.1.15](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.15) (2020-07-20)

### Features
- 默认 init selector

## [0.1.14](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.14) (2020-07-13)

### Features
- 北极星 sdk 版本升级到 v0.5.2

## [0.1.13](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.13) (2020-06-22)

### Features
- 北极星 sdk 版本升级到 v0.5

## [0.1.12](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.12) (2020-05-12)

### Features
- 支持日志目录配置

## [0.1.11](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.11) (2020-04-26)

### Features
- 注册支持权重

## [0.1.10](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.10) (2020-04-10)

### Features
- 北极星 sdk 版本升级到 v0.3.11

## [0.1.9](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.9) (2020-04-01)

### Features
- 修复set 逻辑 panic 
- 北极星 sdk 版本升级到 v0.3.9

## [0.1.8](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.8) (2020-03-26)

### Features
- 支持集群配置
- 支持 set 功能

## [0.1.7](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.7) (2020-03-19)

### Features
- 修复 loadbalance 不带instance 节点的问题

## [0.1.6](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.6) (2020-03-13)

### Features
- 北极星 sdk 版本升级到 v0.3.6

## [0.1.5](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.5) (2020-03-12)

### Features
- 北极星 sdk 版本升级到 v0.3.5
- 支持熔断器配置
- 支持持久化目录配置
- 支持一致性 hash

## [0.1.4](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.4) (2020-03-02)

### Features
- 添加插件统计数据上报

## [0.1.3](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.3) (2020-01-16)

### Features
- 北极星 sdk 版本升级到 v0.3.3
- 移除启动日志

## [0.1.2](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.2) (2020-01-16)

### Features
- 北极星 sdk 版本升级到 v0.3.2 

## [0.1.1](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.1) (2020-01-16)

### Features
- 优化多节点情况下的性能

## [0.1.0](https://git.code.oa.com/trpc-go/trpc-naming-polaris/tree/v0.1.0) (2020-01-13)

### Features
- 支持北极星寻址方式
- 支持多环境功能

