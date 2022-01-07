# 服务注册插件

## 使用方式
```go
import _ "git.code.oa.com/trpc-go/trpc-naming-polaris/registry"
```

## `！注意` 
- 服务注册所需要的 token 和 instance_id 可以从 `http://polaris.oa.com/` 获取
- service name 与上面的sever 的 service 配置一一对应，否则注册失败

## 配置完整示例 (只需要上报心跳)
```yaml
plugins:                                               #插件配置
  registry:                                            #服务注册配置
    polaris:                                           #北极星名字注册服务的配置
      register_self: false                             #是否注册，默认为 false，由 123 平台注册
      heartbeat_interval: 3000                         #名字注册服务心跳上报间隔
      # debug: true                                    #是否开启北极星 sdk 的debug 日志
      service:
        - name:  trpc.test.helloworld.Greeter1    #service name 与上面的service配置一一对应
          namespace: namespace-test1              #环境类型，分正式Production和非正式Development两种类型
          token: 83305443ca420afce9d2             #服务注册所需要的 token
          instance_id: a01e4a596f6d3dc1           # (可选) 服务注册所需要的, instance_id=XXX(namespace+service+host+port)获取摘要
          bind_address: eth1:8080                 # (可选) 指定服务监听地址，默认采用service中的地址
```

## 配置完整示例 (注册 + 上报心跳)
```yaml
plugins:                                               #插件配置
  registry:                                            #服务注册配置
    polaris:                                           #北极星名字注册服务的配置
      register_self: true                              #是否注册，默认为 false，由 123 平台注册
      heartbeat_interval: 3000                         #名字注册服务心跳上报间隔
      # debug: true                                    #是否开启北极星 sdk 的debug 日志
      service:
        - name:  trpc.test.helloworld.Greeter1    #service name 与上面的service配置一一对应
          namespace: namespace-test1              #环境类型，分正式Production和非正式Development两种类型
          token: 83305443ca420afce9d2             #服务注册所需要的 token
          #weight: 100                            #权重默认 100
          #metadata:                              #注册时自定义metadata
          #  key1: val1
          #  key2: val2
```
