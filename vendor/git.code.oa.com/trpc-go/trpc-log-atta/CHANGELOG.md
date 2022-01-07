# Change Log

## [0.1.13](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.13) (2021-08-18)

### Bug Fixes
- 修复log.GetLevel获取atta日志级别报空指针异常

## [0.1.12](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.12) (2021-04-14)

### Bug Fixes
- attaapi切换仓库https://git.code.oa.com/atta/attaapi-go，修复定时器未释放bug

## [0.1.11](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.11) (2021-03-17)

### Bug Fixes
- 修复配置AutoEscape,包含`|`日志错乱的问题

## [0.1.10](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.10) (2020-12-15)

### Features
- 因外部依赖异常导致atta初始化失败不影响服务启动

## [0.1.9](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.9) (2020-10-28)

### Features
- field配置支持远程拉取
- 日志级别支持鹰眼告警

## [0.1.8](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.8) (2020-07-15)

### Bug Fixes
- 修复日志截断时，msg长度不够引发的panic

## [0.1.7](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.7) (2020-07-02)

### Features
- 支持多次WithContextField

## [0.1.6](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.6) (2020-05-22)

### Features
- 支持日志批量上报

## [0.1.5](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.5) (2020-05-13)

### Bug Fixes
- 修复某些情况下[]byte复用引发的日志错乱问题

## [0.1.4](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.4) (2020-05-06)

### Features
- 自定义Encoder

## [0.1.3](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.3) (2020-04-13)

### Features
- 多logger支持

# Change Log

## [0.1.2](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.2) (2020-03-18)

### Bug Fixes
- 解析编译问题，更新trpc-go依赖到v0.2.0

## [0.1.1](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.1) (2020-03-02)

### Features
- 添加插件统计数据上报

## [0.1.0](https://git.code.oa.com/trpc-go/trpc-log-atta/tree/v0.1.0) (2020-01-16)

### Features
- 鹰眼日志
