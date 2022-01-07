# tRPC-Go kafka 插件
[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-430a59b8132c414fbb7a8feed84cb5c0/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com:/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-430a59b8132c414fbb7a8feed84cb5c0/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject)[![Coverage](https://tcoverage.woa.com/api/getCoverage/getTotalImg/?pipeline_id=p-430a59b8132c414fbb7a8feed84cb5c0)](http://macaron.oa.com/api/coverage/getTotalLink/?pipeline_id=p-430a59b8132c414fbb7a8feed84cb5c0)[![GoDoc](https://img.shields.io/badge/API%20Docs-GoDoc-green)](http://godoc.oa.com/git.code.oa.com/trpc-go/trpc-database/kafka)

封装社区的 [sarama](https://github.com/Shopify/sarama) ，配合 trpc 使用。

## producer client
```yaml
client:                                            #客户端调用的后端配置
  service:                                         #针对单个后端的配置
    - name: trpc.app.server.producer               #生产者服务名自己随便定义         
      target: kafka://ip1:port1,ip2:port2?topic=YOUR_TOPIC&clientid=xxx&compression=xxx
      timeout: 800                                 #当前这个请求最长处理时间
```
```go
package main

import (
	"time"
	"context"
	
	"git.code.oa.com/trpc-go/trpc-database/kafka"
	"git.code.oa.com/trpc-go/trpc-go/client"
)

func (s *server) SayHello(ctx context.Context, req *pb.ReqBody, rsp *pb.RspBody)( err error ) {

	proxy := kafka.NewClientProxy("trpc.app.server.producer") // service name自己随便填，主要用于监控上报和寻找配置项

	// kafka 命令
	err := proxy.Produce(ctx, key, value)

	// 业务逻辑
}
```

## consumer service

```yaml
server:                                                                                   #服务端配置
  service:                                                                                #业务服务提供的service，可以有多个
    - name: trpc.app.server.consumer                                                      #service的路由名称 如果使用的是123平台，需要使用trpc.${app}.${server}.consumer  
      address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=x.x.x.x        #kafka consumer broker address，version如果不设置则为1.1.1.0，部分ckafka需要指定0.10.2.0
      protocol: kafka                                                                     #应用层协议 
      timeout: 1000                                                                       #框架配置,与sarama配置无关

```
```go
package main

import (
	"context"
	
	"git.code.oa.com/trpc-go/trpc-database/kafka"
	trpc "git.code.oa.com/trpc-go/trpc-go"
)

func main() {

	s := trpc.NewServer()
	// 启动多个消费者的情况，可以配置多个service，然后这里任意匹配 kafka.RegisterHandlerService(s.Service("name"), handle)，没有指定name的情况，代表所有service共用同一个handler
	kafka.RegisterHandlerService(s, handle) 
	s.Serve()
}

// 只有返回成功nil，才会确认消费成功，返回err不会确认成功，会等待3s重新消费，会有重复消息，一定要保证处理函数幂等性
func handle(ctx context.Context, key, value []byte, topic string, partition int32, offset int64) error {

	return nil
}
```

### 批量消费

```go

import (
	"context"
	
	"git.code.oa.com/trpc-go/trpc-database/kafka"
	trpc "git.code.oa.com/trpc-go/trpc-go"
)

func main() {

	s := trpc.NewServer()
	kafka.RegisterBatchHandlerService(s, handle) 
	s.Serve()
}

// 注意: 必须要配置batch参数(>0), 如果不配置batch参数会发生消费处理函数不匹配导致消费失败
// 完整使用例子参考examples/batchconsumer
// 只有返回成功nil，才会确认消费成功，返回err 整个批次所有消息重新消费
func handle(ctx context.Context, msgArray []*sarama.ConsumerMessage) error {
  // ...
	return nil
}

```


## 参数说明

### 生产者
```yaml
client:
  service:
    - name: trpc.app.server.producer
      target: kafka://ip1:port1,ip2:port2?topic=YOUR_TOPIC&clientid=xxx&compression=xxx
      timeout: 800
```

#### ip:port列表
复数地址使用逗号分割，支持域名:port，ip:port。暂不支持cl5

#### clientid
生产者ID。若为虫洞kafka需要到管理页面注册

#### topic
生产的topic，调用Produce必需。若为虫洞kafka需要到管理页面注册

#### version
客户端版本，支持以下两种格式的版本号
```go
	if s[0] == '0' {
		err = scanKafkaVersion(s, `^0\.\d+\.\d+\.\d+$`, "0.%d.%d.%d", [3]*uint{&minor, &veryMinor, &patch})
	} else {
		err = scanKafkaVersion(s, `^\d+\.\d+\.\d+$`, "%d.%d.%d", [3]*uint{&major, &minor, &veryMinor})
	}
```

#### partitioner
- random 等于sarama.NewRandomPartitioner(默认值)
- roundrobin 等于sarama.NewRoundRobinPartitioner
- hash 等于sarama.NewHashPartitioner
- 暂未自定义hash方式

#### compression
- none 等于sarama.CompressionNone(默认值)
- gzip 等于sarama.CompressionGZIP
- snappy 等于sarama.CompressionSnappy
- lz4 等于sarama.CompressionLZ4
- zstd 等于sarama.CompressionZSTD

#### maxMessageBytes
一次发送的消息最大长度，默认131072

#### requiredAcks
生产消息时，broker返回消息回执（ack）的模式，支持以下3个值（0/1/-1）：

- 0: NoResponse，不需要等待broker响应。
- 1: WaitForLocal，等待本地（leader节点）的响应即可返回。
- -1: WaitForAll，等待所有节点（leader节点和所有In Sync Replication的follower节点）均响应后返回。

默认值是-1。

### 消费者
```yaml
server:
  service:
    - name: trpc.app.server.consumer
      address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=x.x.x.x
      protocol: kafka
      timeout: 1000
```

#### ip:port列表
复数地址使用逗号分割，支持域名:port，ip:port。暂不支持cl5

#### group
消费者组，若为虫洞kafka需要到管理页面注册

#### topics
消费的的toipc，复数逗号分割。

#### compression
- none 等于sarama.CompressionNone(默认值)
- gzip 等于sarama.CompressionGZIP
- snappy 等于sarama.CompressionSnappy
- lz4 等于sarama.CompressionLZ4
- zstd 等于sarama.CompressionZSTD

#### strategy
- sticky 等于sarama.BalanceStrategySticky
- range 等于sarama.BalanceStrategyRange
- roundrobin 等于sarama.BalanceStrategyRoundRobin

#### fetchDefault
默认524288 参照 https://git.code.oa.com/bigdata_fangzhou/MqExample/tree/master/CdmqGoExample

#### fetchMax
默认1048576 参照 https://git.code.oa.com/bigdata_fangzhou/MqExample/tree/master/CdmqGoExample

#### batch
使用批量消费时必填, 注册批量消费函数时, batch不填会发生参数不匹配导致消费失败, 使用参考 examples/batchconsumer

#### batchFlush
默认2秒, 单位ms, 表示当批量消费不满足最大条数时，强制消费的间隔 

#### initial
新消费者组第一次连到集群 消费的位置

#### maxWaitTime
单次消费拉取请求最长等待时间,最长等待时间仅在没有最新数据时才会等待,默认1s

#### maxRetry
失败最大重试次数,超过后直接确认并继续消费下一条消息,默认0:没有次数限制,一直重试

#### netMaxOpenRequests
网络层配置，最大同时请求数, 默认5

#### maxProcessingTime
消费者单条最大请求时间, 单位ms，默认1000ms

#### netDailTimeout
网络层配置，链接超时时间，单位ms，默认30000ms

#### netReadTimeout
网络层配置，读超时时间，单位ms，默认30000ms

#### netWriteTimeout
网络层配置，写超时时间，单位ms，默认30000ms

#### groupSessionTimeout
消费组session超时时间，单位ms，默认10000ms

#### groupRebalanceTimeout
消费者rebalance超时时间，单位ms，默认60000ms

#### mechanism
使用密码时加密方式，SCRAM-SHA-512/SCRAM-SHA-256

#### user 
用户名

#### password 
密码


## 常见问题

- Q1: 消费者的service.name该怎么写
- A1: 如果只有消费者一个service，则名字可以任意起（trpc框架默认会把实现注册到server里面的所有service），完整见examples/consumer
``` yaml
server:                                                                  
  service:                                                               
    - name: trpc.anyname.will.works       #如果使用的是123平台，需要使用trpc.${app}.${server}.consumer                              
      address: 9.134.192.186:9092?topics=test_topic&group=uzuki_consumer 
      protocol: kafka                                                    
      timeout: 1000                                                      
```
``` go
    s := trpc.NewServer()
    kafka.RegisterHandlerService(s, handle) 
```
如果有多个service，则需要在注册时指定与配置文件相同的名字，完整见examples/consumer_with_mulit_service
``` yaml
server:                                                                   
  service:                                                                
    - name: trpc.databaseDemo.kafka.consumer1                             
      address: 9.134.192.186:9092?topics=test_topic&group=uzuki_consumer1 
      protocol: kafka                                                     
      timeout: 1000                                                       
    - name: trpc.databaseDemo.kafka.consumer2                             
      address: 9.134.192.186:9092?topics=test_topic&group=uzuki_consumer2 
      protocol: kafka                                                     
      timeout: 1000     
```
``` go
    s := trpc.NewServer()    
    kafka.RegisterConsumerService(s.Service("trpc.databaseDemo.kafka.consumer1"), &Consumer{})
    kafka.RegisterConsumerService(s.Service("trpc.databaseDemo.kafka.consumer2"), &Consumer{})
```

- Q2: 如果消费时handle返回了非nil会发生什么

- A2: 会休眠3s后重新消费，不建议这么做，失败的应该由业务做重试逻辑

- Q3: 使用ckafka生产消息时，提示
``` log
err:type:framework, code:141, msg:kafka client transport SendMessage: kafka server: Message contents does not match its CRC.
```
- A3: 默认启用了gzip压缩，优先考虑在target上加上参数**compression=none**
``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&compression=none
```

- Q4: 使用ckafka消费消息时，提示
``` log
kafka server transport: consume fail:kafka: client has run out of available brokers to talk to (Is your cluster reachable?)
```
- A4: 优先检查brokers是否可达，然后检查支持的kafka客户端版本，尝试在配置文件address中加上参数例如**version=0.10.2.0**
``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&version=0.10.2.0 
```
- Q5: 消费消息时，提示
``` log
kafka server transport: consume fail:kafka server: The provider group protocol type is incompatible with the other members.
``` 
- A5: 同一消费者组的客户端重分组策略不一样，可修改参数**strategy**，可选：sticky(默认)，range，roundrobin
``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&group=xxx&strategy=range
```
- Q6: 生产时同一用户需要有序，如何配置
- A6: 客户端增加参数**partitioner**，可选random（默认），roundrobin，hash（按key分区）
``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&partitioner=hash
```
- Q7: 如何异步生产
- A7: 客户端增加参数**async=1**
``` yaml
target: kafka://ip1:port1,ip2:port2?clientid=xxx&async=1
```
- Q8: 遇到北极星路由问题 "Polaris-1006(ErrCodeServerError)" "not found service" 
- A8: 确认一下trpc配置中的service.name是  trpc.**${app}.${server}**.AnyUniqNameWillWork 而不是 trpc.**app.server**.AnyUniqNameWillWork, 必须要用占位符
错误现场：
``` log
type:framework, code:131, msg:client Select: get source service route rule err: Polaris-1006(ErrCodeServerError): Response from {ID: 2079470528, Service: {ServiceKey: {namespace: "Polaris", service: "polaris.discover"}, ClusterType: discover}, Address: 9.97.76.132:8081}: not found service
```

- Q9: 如何使用账号密码链接
- A9: 需要在链接参数中配置加密方式、用户名和密码
例如：
``` yaml
address: ip1:port1,ip2:port2?topics=topic1,topic2&mechanism=SCRAM-SHA-512&user={user}&password={password}
```



