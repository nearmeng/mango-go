# Change Log

## [0.1.19](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.18) (2021-12-13)
### Bug Fixes
- 升级SDK版本
- 解决Provider的Watch事件丢失问题

## [0.1.18](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.18) (2021-9-29)
### Bug Fixes
- 升级SDK版本：老版本SDK处理tconf会无法写入缓存

## [0.1.17](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.17) (2021-9-14)
### Bug Fixes
- 解决无缓存的配置拉取bug

## [0.1.16](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.16) (2021-9-10)
### Bug Fixes
- 配置获取不到则返回rainbow的错误
- 调用get接口前先watch一次
- 关闭sdk日志输出

## [0.1.15](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.15) (2021-8-05)
### Bug Fixes
- 每次都给KV配置增加一个Watch操作，保证后台一定能监听配置

## [0.1.14](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.14) (2021-8-02)
### Bug Fixes
- 解决如果没有使用Kv读取的配置，无法监听新配置变更

## [0.1.13](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.13) (2021-7-18)
### Features
- 支持tconf的set配置

## [0.1.12](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.12) (2021-6-30)
### Features
- 支持group数据结构
- 优化table数据组装方式
- 支持拉取 rainbow_tconf配置
- 使用sdk的v3协议

## [0.1.11](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.11) (2021-2-26)
- 升级trpc框架版本

## [0.1.10](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.10) (2021-1-13)
- 升级依赖的SDK版本

## [0.1.9](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.9) (2020-12-30)
- 升级依赖的SDK版本

## [0.1.8](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.8) (2020-12-1)
- 将 addWatcher 失败的报错传给业务层

## [0.1.7](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.7) (2020-11-23)

### Features
- 修复多环境支持bug

## [0.1.6](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.6) (2020-11-9)

### Features
- 支持table数据结构

## [0.1.5](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.5) (2020-11-09)
- 支持 rainbow 多环境配置参数 `env_name`


## [0.1.4](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.4) (2020-09-24)

### Features
- 升级依赖的SDK版本

## [0.1.3](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.3) (2020-04-21)

### Features
- 升级依赖的SDK版本
- 增加SDK拉取配置address、timeout设置

## [0.1.2](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.2) (2020-03-26)

### Features
- 支持开启签名
- 支持设定uin

## [0.1.1](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.1) (2020-03-23)

### Features
- **enable_client_provider** 开启时，插件初始化时会自动拉取配置并应用

## [0.1.0](https://git.code.oa.com/trpc-go/trpc-config-rainbow/tree/v0.1.0) (2020-03-17)

### Features
- 添加插件统计数据上报
- 支持client provider
