# Change Log

## [0.8.0](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.8.0) (2021-11-16)

### Features
- 新增 server client 过载保护模块
- udp service 支持协程池
- udp client transport 支持 buffer 池
- 依赖模块 jce 和 reuseport 切换到 woa
- 优化 metrics histogram

### Bug Fixes
- 解决日志模块 race 问题
- 解决弱依赖插件 bug
- 解决 compress type copy 问题
- 解决无断言单测问题
- 解决 restful 单测偶现失败问题
- 解决 stream client 覆盖 transport 问题

## [0.7.3](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.7.3) (2021-10-14)

### Features
- NoopSerialization Body 支持接口
- server 端空闲时间支持框架配置 server.service.idletime
- 优化连接复用逻辑
- errs 包支持设置跳过堆栈帧数
- 添加日志写入量属性监控 trpc.LogWriteSize
- 添加 trpc.Go(ctx, timeout, handler) 工具函数，方便用户启动异步任务，减少 ctx 相关 bug

### Bug Fixes
- restful 回包没有设置 Content-Type
- plugin 包内的 Config 结构体去除全局变量依赖
- go.mod 去除插件依赖
- 解决单测偶现失败问题
- 解决 http client 没有设置染色消息类型问题

## [0.7.2](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.7.2) (2021-09-06)

### Features
- 支持 flatbuffers
- 连接池支持最小空闲连接数
- restful 支持跨域
- 客户端支持 WithDialTimeout
- RESTful 性能优化并支持设置默认的 Serializer
- 提供公共的安全随机函数可支持多模块调用
- 添加 panic buffer 长度定义
- 添加两个新的框架错误码
  - 23  被服务端限流
  - 123 被客户端限流

### Bug Fixes
- 将多路复用每个连接的队列长度默认值从 100k 改为 1024
- 在 copyCommonMessage 中加上对 commonMeta 和 CompressType 的拷贝
- 多路复用可以正确地返回客户端超时(101)和用户取消(161)两种错误
- 框架 udp 增加 context check
- 修复 m007 上报 RemoteAddr 为空

## [0.7.1](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.7.1) (2021-08-03)

### Features
- 连接池支持初始化连接数
- client 支持 WithLocalAddr Option

### Bug Fixes
- 修复 restful 协议定义绝对路径时的空指针问题
- 修改超时控制有歧义注释
- 修复 msg resetDefault 时没将 callType 重置回默认值的问题
- 一些 typo 修改

## [0.7.0](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.7.0) (2021-07-16)
### Features
- 支持tRPC RESTful，pb option注解生成restful接口
- 支持服务端过载保护
- config接口提供gomock能力
- 支持WriteV系统调用，提升发包效率
- 支持采集上报服务端和客户端包大小


### Bug Fixes
- 修复http客户端无法从错误码判断是否超时
- 修复admin包unregisterHandlers 的数组越界问题
- 修复udp FramerBuilder 为 nil 错误
- 优化相同配置文件变更事件只触发一次
- 修复流式服务端error没有返回给客户端 
- admin调整为service实现，避免独立客户端无法开启pprof问题
- 修复多路复用重复 close 导致 err 变更问题

## [0.6.6](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.6.6) (2021-06-25)

### Features
- 性能优化
- 支持只发不收
- 更新godoc到pkg.woa.com

### Bug Fixes
- 解决连接泄露问题
- 解决内存占用大问题
- 解决rand.Seed干扰问题

## [0.6.5](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.6.5) (2021-05-27)

### Features
- 性能优化：slice预分配内存
- 提升连接池空闲状态检查时效性
- udp client校验framer
- 插件支持弱依赖关系
- errs堆栈支持过滤能力
- http client支持 patch 方法

### Bug Fixes
- 解决单测偶现失败问题
- 解决http transinfo env-key base64问题
- 解决client stream data race问题

## [0.6.4](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.6.4) (2021-05-13)

### Bug Fixes
- 解决registry检查失败问题
- 流式关闭连接导致decode错误

## [0.6.3](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.6.3) (2021-05-12)

### Features
- 性能优化：协程池改为开源ants实现
- http status code支持2xx成功返回码

### Bug Fixes
- udp解包失败直接丢包，解决udp server和dns server冲突问题
- http transinfo env-key base64编码
- selector options loadbalancer拼写错误问题
- 多路复用失败重连

## [0.6.2](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.6.2) (2021-04-26)

### Features
- 支持http post multipart/form-data
- 尽早设置http client rsp header

### Bug Fixes
- 解决包长度溢出bug
- 解决单测偶发失败问题
- 解决代码规范问题
- 修复向已关闭的stream流写入时不会返回err

## [0.6.1](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.6.1) (2021-04-16)

### Bug Fixes
- 解决http clent request content-length为0问题

## [0.6.0](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.6.0) (2021-04-14)

### Features
- 支持udp client transport io复用
- 支持服务无损更新
- 支持http/https客户端链接参数设置
- client在拦截器之前设置超时时间
- 性能优化

### Bug Fixes
- 解决http client大包内存泄露问题
- 解决代码重复问题
- 解决流式无法获取metadata问题
- 解决单测偶现失败问题

## [0.5.2](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.5.2) (2021-02-26)

### Features
- 统一收拢trpc工具类函数到trpc_util.go文件
- 统一收拢环境变量key到internal/env/env.go文件
- 统一收拢监控上报key到internal/report/metrics_report.go文件
- 去除重定向std log到日志文件逻辑，提供`log.RedirectStdLog`函数供用户调用
- 流控功能实现完成
- 支持动态设置虚拟节点数
- 支持同时使用有协议和无协议http服务
- admin使用`net/http/pprof`，支持分析cpu，内存
- 支持配置`network: tcp,udp`同时监听tcp和udp
- http支持`application/xml`

### Bug Fixes
- 解决client.DefaultClientConfig并发问题
- 解决http env多环境透传问题
- 解决创建日志实例失败导致panic问题
- 解决client target非域名解析卡顿问题
- 解决io复用内存泄露问题
- 禁用服务路由时清空多环境透传信息
- 解决client后端拦截器并发问题

## [0.5.1](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.5.1) (2021-01-08)

### Features
- 增加trpc.CloneContext接口，方便异步处理
- 增加client.WithMultiplexedPool接口，方便用户自定义io复用连接参数
- 增加config.Reload接口

### Bug Fixes
- 日志按时间滚动也限制大小，异步满丢弃上报监控
- 优化大包时，内存使用率过高问题
- 解决圈复杂度超标问题
- 修复DataFrameType字段错误问题

## [0.5.0](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.5.0) (2020-12-28)

### Features
- 支持client重试策略
- 性能优化：支持协程池
- 性能优化：gzip压缩缓存
- 性能优化：io复用支持多连接
- 支持http application/x-protobuf Content-Type
- WithTarget支持负载均衡方式
- http client transport支持配置最大空闲连接数
- selector支持传入context

### Bug Fixes
- 日志模式默认极速写：日志异步写队列，队列满则丢弃
- 修复client filter获取不到请求header问题
- 解决代码规范问题，圈复杂度超标问题
- 更新覆盖率图标到https链接，解决chrome mixed-content问题
- 解决filter非并发安全问题

## [0.4.2](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.4.2) (2020-11-26)

### Bug Fixes
- 框架配置解析环境变量，只解析${var}不解析$var，解决redis密码包含$字符问题

## [0.4.1](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.4.1) (2020-11-24)

### Bug Fixes
- 修复kafka等自定义协议没配置ip的情况

## [0.4.0](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.4.0) (2020-11-24)

### Features
- 支持流式
- 客户端连接模式支持IO复用
- 单测覆盖率提升到87%以上
- Config接口支持toml格式
- Config支持填写默认值
- client寻址逻辑移到拦截器内部
- 框架配置支持环境变量占位符

### Bug Fixes
- admin模块去掉net/http/pprof依赖，解决安全问题
- 修复.code.yml问题
- 修复client配置timeout不生效问题
- 解决代码规范问题，圈复杂度过高问题
- 解决框架配置nic填错没有阻止启动问题
- http响应没有返回透传字段trpc-trans-info

## [0.3.7](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.3.7) (2020-09-22)

### Features
- errs增加WithStack携带调用栈信息
- 热重启信号量更改为变量允许用户自己修改
- 服务端默认异步server_async

### Bug Fixes
- 解决热重启问题
- 解决http response error msg错误问题
- noresponse不关闭连接

## [0.3.6](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.3.6) (2020-07-29)

### Features
- http client method支持option参数
- 框架自身监控上报属性加上trpc.前缀
- 支持单个client配置set_name env_name disable_servicerouter

### Bug Fixes
- 解决连接池复用bug，导致串包问题
- 解决log删除多余备份失效问题
- 解决http rpcname invalid问题
- 解决多维监控无法设置name问题

## [0.3.5](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.3.5) (2020-07-27)

### Bug Fixes
- 解决框架SetGlobalConfig后移导致插件启动失败问题
- 修复client namespace为空问题

## [0.3.4](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.3.4) (2020-07-24)

### Features
- rpc invalid时，添加当前服务service name，方便排查问题
- 提高单测覆盖率
- http端口443时默认设置schema为https
- 将开源lumberjack日志切换为内置rollwriter日志，提高打日志性能
- 解决圈复杂度问题，每个函数尽量控制到5以内
- 对端口复用的httpserver添加热重启时停止接收新请求

### Bug Fixes
- 解决动态设置日志等级无效问题
- 修复同一server使用多个证书时缓存冲突问题
- 修复http client 连接失败上报问题
- 解决server write 错误导致死循环问题
- 解决server代理透传二进制问题
- 解决http get请求无法解析二进制字段问题
- 解决框架启动调用两次SetGlobalConfig问题

## [0.3.3](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.3.3) (2020-07-01)

### Features
- http default transport使用原生标准库的default transport
- 支持client短连接模式
- 支持设置自定义连接池
- 日志key字段支持配置
- 连接池MaxIdle最大连接数调整为无上限

### Bug Fixes
- 解决server filter去重问题
- 解决ip硬编码安全规范问题
- 解决代码规范问题

## [0.3.2](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.3.2) (2020-06-18)

### Features
- 支持server端异步处理请求，解决非trpc-go client调用超时问题
- 框架内部默认import uber automaxprocs，解决容器内调度延迟问题

### Bug Fixes
- 解决client filter覆盖清空问题
- 解决http server CRLF注入问题

## [0.3.1](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.3.1) (2020-06-10)

### Features
- 支持用户自己设置Listener
- 支持http get请求独立序列化方式

### Bug Fixes
- 解决client filter执行两次的问题 
- 解决server回包无法指定序列化方式和压缩方式问题
- 解决http client proxy用户无法设置protocol的问题

## [0.3.0](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.3.0) (2020-05-29)

### Features
- 支持传输层tls鉴权
- 支持http2 protocol
- 支持admin动态设置不同logger不同output的日志等级
- 支持http Put Delete方法

## [0.2.8](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.8) (2020-05-12)

### Features
- 代码OWNER制度更改，owners.txt改成.code.yml，符合epc标准
- 支持http client post form请求
- 支持client SendOnly只发不收请求
- 支持自定义http路由mux
- 支持http.SetContentType设置http content-type到trpc serialization type的映射关系，兼容不规范老http框架服务返回乱写的content-type

### Bug Fixes
- 解决http client rsp没有反序列化问题
- 解决tcp server空闲时间不生效问题
- 解决多次调用log.WithContextFields新增字段不生效问题

## [0.2.7](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.7) (2020-04-30)

### Bug Fixes
- 解决flag启动失败问题

## [0.2.6](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.6) (2020-04-29)

### Features
- 复用msg结构体，echo服务性能从39w/s提升至41w/s
- 提升单元测试覆盖率至84.6%
- 新增一致性哈希路由算法

### Bug Fixes
- tcp listener没有close
- 解决NewServer flag定义冲突问题

## [0.2.5](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.5) (2020-04-20)

### Features
- 添加trpc.NewServerWithConfig允许用户自定义框架配置文件格式
- 支持https client，支持https双向认证
- 支持http mock
- 添加性能数据实时看板，readme benchmark icon入口

### Bug Fixes
- 将所有gogo protobuf改成官方的golang protobuf，解决兼容问题
- admin启动失败直接panic，解决admin启动失败无感知问题

## [0.2.4](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.4) (2020-04-02)

### Features
- http server head 添加原始包体ReqBody
- 配置文件支持toml序列化方式
- 添加client CalleeMethod option，方便自定义监控方法名
- 添加dns寻址方式：dns://domain:port

### Bug Fixes
- 改造log api，将Warning改成Warn
- 更改DefaultSelector为接口方式

## [0.2.3](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.3) (2020-03-24)

### Bug Fixes
- 禁用client filter时不加载filter配置

## [0.2.2](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.2) (2020-03-23)

### Features
- 框架内部关键错误上报metrics
- 多维监控使用数组形式

## [0.2.1](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.1) (2020-03-19)

### Features
- 支持禁用client拦截器

## [0.2.0](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.2.0) (2020-03-18)

### Bug Fixes
- 解决golint问题

### Features
- 支持set路由
- client config支持配置下游的序列化方式和压缩方式
- 框架支持metrics标准多维监控接口
- 所有wiki文档全部转移到iwiki

## [0.1.6](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.6) (2020-03-11)

### Bug Fixes
- 新增插件初始化完成事件通知

## [0.1.5](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.5) (2020-03-09)

### Bug Fixes
- 解决golint问题
- 解决client transport收包失败都返回101超时错误码问题

### Features
- client transport framer复用
- http server decode失败返回400，encode失败返回500
- 新增更安全的多并发简易接口 trpc.GoAndWait 
- 新增http client通用的Post Get方法
- server拦截器未注册不让启动
- 日志caller skip支持配置
- 支持https server
- 添加上游客户端主动断开连接，提前取消请求错误码 161

## [0.1.4](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.4) (2020-02-18)

### Bug Fixes
- 客户端设置不自动解压缩失效问题

## [0.1.3](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.3) (2020-02-13)

### Bug Fixes
- 插件初始化加载bug

## [0.1.2](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.2) (2020-02-12)

### Bug Fixes
- http client codec CalleeMethod覆盖问题
- server/client mock api失效问题

### Features
- 新增go1.13错误处理error wrapper模式
- 添加插件初始化依赖顺序逻辑
- 新增trpc.BackgroundContext()默认携带环境信息，避免用户使用错误

## [0.1.1](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.1) (2020-01-21)

### Bug Fixes
- http client transport无法设置content-type问题
- 天机阁ClientFilter取不到CalleeMethod问题
- http client transport无法设置host问题

### Features
- 增加disable_request_timeout配置开关，允许用户自己决定是否继承上游超时时间，默认会继承
- 增加callee framework error type，用以区分当前框架错误码，下游框架错误码，业务错误码
- 下游超时时，errmsg自动添加耗时时间，方便定位问题
- http server回包header增加nosniff安全header
- http被调method使用url上报


## [0.1.0](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0) (2020-01-10)

### Bug Fixes
- 滚动日志默认按大小，流水日志按日期
- 日志路径和文件名拼接bug
- 指定环境名路由bug

### Features
- 代码格式优化，符合epc标准
- 插件上报统计数据

## [0.1.0-rc.14](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.14) (2020-01-06)

### Bug Fixes
- 连接池默认最大空闲连接数过小导致频繁创建fd，出现timewait爆满问题，改成默认MaxIdle=2048
- server transport没有framer builder导致请求crash问题

### Features
- 支持从名字服务获取被调方容器名

## [0.1.0-rc.13](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.13) (2019-12-30)

### Bug Fixes
- 连接池偶现EOF问题：server端统一空闲时间1min，client端统一空闲时间50s
- 高并发下超时设置http header crash问题：去除service select超时控制
- http回包json enum变字符串 改成 enum变数字，可配置
- http header透传信息二进制设置失败问题，改成transinfo base64编码

### Features
- 支持无协议文件自定义http路由
- 支持请求http后端携带header
- http服务支持reuseport热重启


## [0.1.0-rc.12](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.12) (2019-12-24)

### Bug Fixes
- 包大小uint16限制
- metrics counter锁bug
- 单个插件初始化超时3s，防止服务卡死
- 同名网卡ip覆盖
- 多logger失效

### Features
- 指定环境名路由
- http新增自定义ErrorHandler
- timer改成插件模式
- 添加godoc icon


## [0.1.0-rc.11](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.11) (2019-12-09)

### Bug Fixes
- udp client transport对象池复用导致buffer错乱


## [0.1.0-rc.10](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.10) (2019-12-05)

### Bug Fixes
- udp client connected模式writeto失败问题



## [0.1.0-rc.9](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.9) (2019-12-04)

### Bug Fixes
- 连接池超时控制无效
- 单测偶现失败
- 默认配置失效

### Features
- 新增多环境开关
- udp client transport新增connection mode，由用户自己控制请求模式
- udp收包使用对象池，优化性能
- admin新增性能分析接口


## [0.1.0-rc.8](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.8) (2019-11-26)

### Bug Fixes
- server WithProtocol option漏了transport
- 后端回包修改压缩方式不生效
- client namespace配置不生效

### Features
- 支持 client工具多环境路由
- 支持 admin管理命令
- 支持 热重启
- 优化 日志打印


## [0.1.0-rc.7](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.7) (2019-11-21)

### Features
- 支持client option设置多环境


## [0.1.0-rc.6](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.6) (2019-11-20)

### Bug Fixes
- 支持一致性哈希路由


## [0.1.0-rc.5](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.5) (2019-11-08)

### Bug Fixes
- tconf api
- transport空指针bug

### Features
- 多环境治理
- 代码质量管理owner机制


## [0.1.0-rc.4](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.4) (2019-11-04)

### Bug Fixes
- frame builder 魔数校验，最大包限制默认10M

### Features
- 提高单测覆盖率


## [0.1.0-rc.3](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.3) (2019-10-28)

### Bug Fixes
- http client codec

## [0.1.0-rc.2](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.2) (2019-10-25)

### Bug Fixes
- windows 连接池bug

### Features
- 测试覆盖率提高到83%

## [0.1.0-rc.1](https://git.code.oa.com/trpc-go/trpc-go/tree/v0.1.0-rc.1) (2019-10-25)

### Features
- 一发一收应答式服务模型
- 支持 tcp udp http 网络请求
- 支持 tcp连接池，buffer对象池
- 支持 server业务处理函数前后链式拦截器，client网络调用函数前后链式拦截器
- 提供trpc代码[生成工具](https://git.code.oa.com/trpc-go/trpc-go-cmdline)，通过 protobuf idl 生成工程服务代码模板
- 提供[rick统一协议管理平台](http://trpc.rick.oa.com/rick/pb/list)，tRPC-Go插件通过proto文件自动生成pb.go并自动push到[统一git](https://git.code.oa.com/trpcprotocol)
- 插件化支持 任意业务协议，目前已支持 trpc，[tars](https://git.code.oa.com/trpc-go/trpc-codec/tree/master/tars)，[oidb](https://git.code.oa.com/trpc-go/trpc-codec/tree/master/oidb)
- 插件化支持 任意序列化方式，目前已支持 protobuf，jce，json
- 插件化支持 任意压缩方式，目前已支持 gzip，snappy
- 插件化支持 任意链路跟踪系统，目前已使用拦截器方式支持 [天机阁](https://git.code.oa.com/trpc-go/trpc-opentracing-tjg) [jaeger](https://git.code.oa.com/trpc-go/trpc-opentracing-jaeger)
- 插件化支持 任意名字服务，目前已支持 [老l5](https://git.code.oa.com/trpc-go/trpc-selector-cl5)，[cmlb](https://git.code.oa.com/trpc-go/trpc-selector-cmlb)，[北极星测试环境](https://git.code.oa.com/trpc-go/trpc-naming-polaris) 
- 插件化支持 任意监控系统，目前已支持 [老sng-monitor-attr监控](https://git.code.oa.com/trpc-go/metrics-plugins/tree/master/attr)，[pcg 007监控](https://git.code.oa.com/trpc-go/metrics-plugins/tree/master/m007)
- 插件化支持 多输出日志组件，包括 终端console，本地文件file，[远程日志atta](https://git.code.oa.com/trpc-go/trpc-log-remote-atta)
- 插件化支持 任意负载均衡算法，目前已支持 roundrobin weightroundrobin
- 插件化支持 任意熔断器算法，目前已支持 北极星熔断器插件
- 插件化支持 任意配置中心系统，目前已支持 [tconf](https://git.code.oa.com/trpc-go/config-tconf)

### 压测报告

| 环境 | server | client | 数据 | tps | cpu |
| :--: | :--: |:--: |:--: |:--: |:--: |
| 1 | v8虚拟机 9.87.179.247 | 星海平台jmeter 9.21.148.88 | 10B的echo请求 | 25w/s | null |
| 2 | b70物理机 100.65.32.12 | 星海平台jmeter 9.21.148.88 | 10B的echo请求 | 42w/s | null |
| 3 | v8虚拟机 9.87.179.247 | eab工具，b70物理机 100.65.32.13 | 10B的echo请求 | 35w/s | 64% |
| 4 | b70物理机 100.65.32.12 | eab工具，b70物理机 100.65.32.13 | 10B的echo请求 | 60w/s | 45% |

### 测试报告
- 整体单元测试[覆盖率80%](http://devops.oa.com/console/pipeline/pcgtrpcproject/p-da0d17b2016f404fa725983ae020ed01/detail/b-5ee497f8d96348359b874ec062795ca5/output)
- 支持 [server mock能力](https://git.code.oa.com/trpc-go/trpc-go/tree/master/server/mockserver)
- 支持 [client mock能力](https://git.code.oa.com/trpc-go/trpc-go/tree/master/client/mockclient)

### 开发文档
- 每个package有[README.md](https://git.code.oa.com/trpc-go/trpc-go/tree/master/server)
- [examples/features](https://git.code.oa.com/trpc-go/trpc-go/tree/master/examples/features)有每个特性的代码示例
- [examples/helloworld](https://git.code.oa.com/trpc-go/trpc-go/tree/master/examples/helloworld)具体工程服务示例
- [git wiki](https://git.code.oa.com/trpc-go/trpc-go/wikis/home)有详细的设计文档，开发指南，FAQ等

### 下一版本功能规划
- 服务性能优化，提高tps
- 完善开发文档，提高易用性
- 完善单元测试，提高测试覆盖率
- 支持[更多协议](https://git.code.oa.com/trpc-go/trpc-codec)，打通全公司大部分存量平台框架
- admin命令行系统
- auth鉴权
- 多环境/set/idc/版本/哈希 路由能力
- 染色key能力
