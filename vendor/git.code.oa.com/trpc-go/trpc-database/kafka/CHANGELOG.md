# Change Log

## [0.1.8](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.7) (2021-04-27)
### Bug Fixes 
- 修复kafka参数解析问题

## [0.1.7](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.7) (2021-04-16)
### Features
- 支持账号密码配置
- 开放各层级超时时间配置
- 开放网络层参数配置选项

### Bug Fixes 
- 修复kafka异步配置未赋值bug

## [0.1.6](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.6) (2021-03-01)
### Features
- 生产者支持requiredAcks配置
- 增加scram安全认证功能 
- 各配置字段值支持特殊字符,字段值解析增加了url decode逻辑 

## [0.1.5](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.5) (2020-11-27)
### Features
- 支持更多配置参数，补充参数文档
- Client支持Headers 
- 支持批量消费 

## [0.1.4](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.4) (2020-04-10)
### Bug Fixes 
- MockClient支持Produce

## [0.1.3](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.3) (2020-04-07)
### Bug Fixes 
- 修复target相同的时候使用SendMessage/AsyncSendMessage传入不同topic无效
- 修复取不到业务侧返回的err

## [0.1.2](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.2) (2020-03-20)
### Bug Fixes 
- 取消设置sarama.MaxResponseSize，解决同一进程无法同时生产和消费

## [0.1.1](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.1) (2020-03-19)
### Bug Fixes 
- 设置client get producer timeout，解决超时不生效问题

## [0.1.0](https://git.code.oa.com/trpc-go/trpc-database/tree/kafka/v0.1.0) (2020-03-18)
### Features
- 支持kafka producer consumer
- 支持配置dsn详细参数
- 支持配置topic async
