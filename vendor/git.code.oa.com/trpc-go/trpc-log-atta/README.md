[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-b5bc8131706b40b6a639555b7b18770d/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com:/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-b5bc8131706b40b6a639555b7b18770d/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject)[![Coverage](https://tcoverage.woa.com/api/getCoverage/getTotalImg/?pipeline_id=p-b5bc8131706b40b6a639555b7b18770d)](http://macaron.oa.com/api/coverage/getTotalLink/?pipeline_id=p-b5bc8131706b40b6a639555b7b18770d)
# tRPC-Go atta远程日志插件


## 配置

* 完整配置
```yaml
plugins:
  log:                                              #日志配置 支持多个日志 可通过 log.Get("xxx").Debug 打日志
    default:                                        #默认日志的配置，每个日志可支持多输出
      - writer: atta                                #atta远程日志输出
        level: debug                                #远程日志的级别
        remote_config:                              #远程日志配置，业务自定义结构，每一种远程日志都有自己独立的配置
          atta_id: '05e00006180'                    #atta id 每个业务自己申请. 使用之前需要保证机器有安装atta agent
          atta_token: '6851146865'                  #atta token 业务自己申请
          auto_escape: false                        #[可选,默认false] atta上报内容是否转义,性能原因默认关闭,true开启
          attaobj_size: 3                           #[可选,默认3] 日志异步上报,消费log的goroutinue数,一个goroutinue一个att obj,避免竞态
          channel_block: false                      #[可选，默认false] 日志异步上报,写管道是否阻塞.
          channel_capacity: 10000                   #[可选，默认10000] 日志管道容量
          enable_batch: false                       #[可选，默认false] 是否缓存批量发送日志，
          send_internal: 1000                       #[可选，默认1000] enable_batch为true时有效，缓存批量发送时间间隔，单位ms
          message_key: msg                          #日志打印包体的对应atta的field
          level_key: level                          #[可选，默认为空]日志级别对应的atta字段，不需可不配置
          atta_warning: false                       #[可选，默认false]日志级别映射，鹰眼告警开启
          field:                                    #[可选，不配置从远端拉取]申请atta id时，业务自己定义的表结构字段，顺序必须一致
            - msg
            - uid
            - cmd
            - level
```
除了以上常用字段外，用户可以自己申请更详细的字段配置即可，跟message_key规则一致。
框架支持的日志上下文字段有
level_key 日志等级
time_key 时间
caller_key  文件和行号
stacktrace_key 调用栈

* 最小配置
```yaml
plugins:
  log:                                              #日志配置 支持多个日志 可通过 log.Get("xxx").Debug 打日志
    default:                                        #默认日志的配置，每个日志可支持多输出
      - writer: atta                                #atta远程日志输出
        level: debug                                #远程日志的级别
        remote_config:                              #远程日志配置，业务自定义结构，每一种远程日志都有自己独立的配置
          atta_id: '05e00006180'                    #atta id 每个业务自己申请. 使用之前需要保证机器有安装atta agent
          atta_token: '6851146865'                  #atta token 业务自己申请
          message_key: msg                         #日志打印包体的对应atta的field  
```
除了以上常用字段外，用户可以自己申请更详细的字段配置即可，跟message_key规则一致，如
level_key 日志等级
time_key 时间
caller_key  文件和行号
stacktrace_key 调用栈

## 如何使用

#### 1 申请atta id，atta token，创建表结构
* 网址：http://atta.pcg.com/#/dataaccess/frontAccess
* atta id和token在申请审批通过后即可查看
* 首先要确保机器上已经安装了atta agent，如果没有安装，可通过这个[织云包](http://yun.isd.com/index.php/package/versions/?product=other&package=AttaAgent)安装
* 123平台所有容器都已预装好atta agent，无需担心
* 注意：atta udp接口限制， 日志(所有字段长度和)最大长度64K，超过会截断message_key对应内容

#### 2 配置trpc_go.yaml

#### 3 开发代码
- 首先需要import本仓库地址
```golang
    import _ "git.code.oa.com/trpc-go/trpc-log-atta"
```
- atta远程日志跟请求上下文相关，所以需要使用context接口打日志，每个请求入口先调用 WithFields 设置表字段，输入必须是key-value成对出现。
```golang
    log.WithContextFields(ctx, "uid", "11111", "cmd", "/trpc.test.helloworld.Greeter/SayHello") // 已支持多行分开设置，不过每次WithContextFields zap内部logger会clone一次，有一定成本。
    log.DebugContext(ctx, "message1") //这里的""字符串会上报至配置里的message_key字段中
    log.InfoContext(ctx, "message2")
    log.ErrorContext(ctx, "message3")
```
- 多logger的情况没有Context接口，需要自己在请求入口WithField生成新的logger才能打印，多logger的配置与注册请参考
[文档](https://git.code.oa.com/trpc-go/trpc-go/tree/master/log)
```golang
    logger := log.Get("custom").WithFields("uid", "11111", "cmd", "/trpc.test.helloworld.Greeter/SayHello")
    logger.Debug("message1")
    logger.Info("message2")
    logger.Error("message3")
```

- 如果要上报更多的字段，先在配置里添加，然后调用log.WithContextFields(ctx, "key1", "value1")
        
- 由于每个请求入口都需要WithContextFields，所以这种重复性工作可以使用server拦截器来实现
```golang
    import "git.code.oa.com/trpc-go/trpc-go/filter"
```
```golang
    // main.go文件，启动前定义atta拦截器，字段业务自己定。也可以定义在独立的git上，注册到filter，其他服务引用进来，配置使用即可。
    var attaFieldFilter = func(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) error {
        
        msg := trpc.Message(ctx)  
    
        log.WithContextFields(ctx, "uid", msg.DyeingKey(), "cmd", msg.ServerRPCName())
        
        return handler(ctx, req, rsp)
    }  
```
```golang
    func main() {
    
        s := trpc.NewServer(server.WithFilter(attaFieldFilter))
    
        pb.RegisterGreeterServer(s, &GreeterServerImpl{})
    
        s.Serve()
    }
```
#### 4 查看远程日志
- atta本质上是一个数据通道，查看数据要去[鹰眼](http://log2.webdev.com/mylog/logList)(也可先在[这里](http://atta.pcg.com/#/dataManage/myData/realData?attaid=05e00006180)查看atta数据通道是否正常)
- 鹰眼与attaID绑定申请：http://log2.webdev.com/join/joinTab#
- 鹰眼的日志类型选择 【通用ATTA上报日志】
- 申请通过后，即可在【我的日志】里查询

## 使用注意

#### 1 定位
鹰眼日志插件！日志插件！！ atta上报数据需求请直接用atta api: https://git.code.oa.com/atta/attaapi-go

#### 2 转义
auto_escape 配置默认不转义，提升log性能，特殊字符 `\r`、`\n`、`\\` 会显示原始内容，比如换行，不会丢失数据。
但要特别注意atta分隔符 `|`， 会导致log内容错乱。不能接受，请手动开启转义。

#### 3 异步
log异步上报，相关参数：
- attaobj_size 控制异步消费的goroutinue数量，默认为3
- channel_block 控制写管道是否阻塞，默认配置不阻塞。不建议修改：开启本地日志且level小于等于attlog或者有部分业务逻辑，log消费速度基本足够，观察有丢失可考虑增加attaobj_size数量

#### 4 批量发送
enable_batch 日志是否批量发送，默认关闭。

对attaagent版本有要求，atta配置文件atta_agent.cfg里面的agent-version 为1.6或更高.
123平台已全量升级

#### 5 field远端拉取
在field未配置的情况下，支持从远端拉取atta字段列表。若初始化不想依赖过多外部服务，可手动配置。

若远端拉取，svr初始化必失败的话，说明机器不能解析atta接口域名（http://atta.wsd.com/cgi/dataapi）

推荐以下两种方式
- 修复机器DNS配置，参考[UDNS](https://iwiki.woa.com/pages/viewpage.action?pageId=142880738)， 123平台验证已支持
- atta字段不会经常变动，也可手动配置field
