# tRPC-Go rainbow 七彩石配置中心
[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-d9a21fc84f5545088be36f9545ca6de7/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](http://devops.oa.com:/ms/process/api-html/user/builds/projects/pcgtrpcproject/pipelines/p-d9a21fc84f5545088be36f9545ca6de7/latestFinished?X-DEVOPS-PROJECT-ID=pcgtrpcproject)[![Coverage](https://tcoverage.woa.com/api/getCoverage/getTotalImg/?pipeline_id=p-d9a21fc84f5545088be36f9545ca6de7)](http://macaron.oa.com/api/coverage/getTotalLink/?pipeline_id=p-d9a21fc84f5545088be36f9545ca6de7)[![GoDoc](https://img.shields.io/badge/API%20Docs-GoDoc-green)](http://godoc.oa.com/git.code.oa.com/trpc-go/trpc-config-rainbow)

请访问配置中心 Web端控制台(http://rainbow.oa.com/)操作


### 如何使用

#### 0. 平台配置操作

1. 访问Web端控制台(http://rainbow.oa.com/)

2. 新建项目（如果已有项目，跳过），此时浏览器URL中间一串即为插件配置中需要的`appid`字段，如：http://rainbow.oa.com/console/3482e0a7-3a00-401c-9505-7bdb0a12511c/list 中的appid为`3482e0a7-3a00-401c-9505-7bdb0a12511c`

3. 新建分组（如果已有分组，跳过），分组名即为插件配置中的`group`字段

4. 新增配置，并发布配置


#### 1. 增加插件配置
请在框架配置文件 `trpc_go.yaml` 中增加对应的插件配置

```yaml
plugins:
    config:
        rainbow: # 七彩石配置中心
            providers:
              - name: rainbow # provider名字，代码使用如：`config.WithProvider("rainbow")`
                appid: 3482e0a7-3a00-401c-9505-7bdb0a12511c # appid
                group: dev # 配置所属组
                type: kv   # 七彩石数据格式, kv(默认), table, group
                env_name: production
                file_cache: /tmp/a.backup
                uin: a3482e0a7
                enable_sign: true
                user_id: 2a9a63844fe24a8aadaxxx5d2f5e903a
                user_key: 599dd5a3480805e22bb6ac22eeaf40d34f8a
                enable_client_provider: true
                timeout: 2000

              - name: rainbow1
                appid: 3482e0a7-3a00-401c-9505-7bdb0a12511c
                group: dev1

```
###### 配置解析
provider 表示配置所属项目的分组，插件支持从多个 provider 中拉取配置。

- name: provider标识，可以使用：`config.WithProvider("tconf1")`，指定从某个 provider 中拉取配置

- appid: 配置所属的项目

- group: 配置所属的分组

- type: 七彩石数据格式, kv(默认), table, group

- env_name: rainbow多环境配置，如果没有使用多环境特性，不需要配置此项

- timeout: 拉取配置接口超时设置，单位毫秒，不填默认2秒

- address: rainbow服务端地址，内网无需填写，外网使用请咨询: Rainbow_Helper(七彩石运营助手)

- uin: 客户端标识，可选配置

- file_cache: 本地缓存文件设置，可选配置

- enable_sign: 设置签名校验，可选配置，开启时需要设置 `user_id、user_key`

- user_id: 用户ID，平台生成。在拉取配置时生成签名，`enable_sign: true`时 必填

- user_key : 用户密钥，平台生成。在拉取配置时生成签名，`enable_sign: true`时 必填

- enable_client_provider: 使用client provider，可不填，默认为False。 当设置为 `True` 时，会自动从当前的 Provider中监听`client.yaml`配置，并自动更新替换远程调用信息。需要手动在 rainbow 平台中添加`client.yaml`配置。支持的配置可参考[ClientConfig](https://git.code.oa.com/trpc-go/trpc-go/blob/master/config.go#L67)，或参考trpc_go.yaml中的[client配置](https://git.code.oa.com/trpc-go/trpc-go/blob/master/testdata/trpc_go.yaml#L37)，下面是一个简单的`client.yaml`配置示例：
```yaml
    client:
      service:
        - name: trpc.test.helloworld.Greeter1
          discovery: etcd
          namespace: Development
          network: tcp
          protocol: trpc
          timeout: 800
        - name: trpc.test.helloworld.Greeter2
          namespace: Production
          network: tcp
          protocol: http
          target: "l5://11111:222222"
          timeout: 2000
      timeout: 1000
```
#### 使用enable_client_provider 注意
- 只支持在**最多一个** provider 中设置 `enable_client_provider: true`
- 开启设置后，会自动覆盖框架配置（trpc_go.yaml）中 client配置，因此在 rainbow 平台上的 `client.yaml` 配置，**需要配全所需配置**
- 在指定的**项目**的**分组**下增加配置： key填写 `client.yaml`，类型选择 `YAML`，Value填写你需要的配置


#### 如何启用配置拉取签名 (enable_sign: true)

开启步骤： 选择项目 --> 基本信息 --> 启用配置拉取签名

##### err: signature not found
rainbow平台开启签名校验，本地没有配置签名

##### err: signature expire
发送请求服务器的时间和rainbow服务端时间差超过2分钟


#### 2. 插件注册

```go
import (
    // 根据插件配置自动注册ranbow插件
	_ "git.code.oa.com/trpc-go/trpc-config-rainbow"
)

```

#### 3. 插件系统初始化
**所有配置读取操作需要在`trpc.NewServer()`之后**

```go
import (
    // 引入tprc-go
    trpc "git.code.oa.com/trpc-go/trpc-go"
)

// 实例化server时会初始化插件系统，所有配置读取操作需要在此之后
trpc.NewServer()

```

#### 4. 读取配置

```go
import (
	"git.code.oa.com/trpc-go/trpc-go/config"
)

...

// 使用trpc-go/config中新接口读取test.yaml配置
type yamlFile struct {
    Server struct {
        App string
    }
}

yf := &yamlFile{}
// 读取配置并反序列化到yf中
err := config.GetYAML("test.yaml", yf)

...


// 加载provider名为`rainbow`的插件配置，读取`test.yaml`的配置文件，使用`yaml`解析
c, err := config.Load("test.yaml", config.WithCodec("yaml"), config.WithProvider("rainbow"))
appname := c.GetString("server.app", "default")

```

##### Watch 远程配置变化

```go
import (
	"sync/atomic"
    ...

)

// 参考：https://golang.org/pkg/sync/atomic/#Value
var cfg atomic.Value // 并发安全的Value

// 使用trpc-go/config中Watch接口监听tconf远程配置变化
c, _ := config.Get("rainbow").Watch(context.TODO(), "test.yaml")

go func() {
    for r := range c {
        yf := &yamlFile{}
        fmt.Printf("event: %d, value: %s", r.Event(), r.Value())

        if err := yaml.Unmarshal([]byte(r.Value()), yf); err == nil {
            cfg.Store(yf)
        }

    }
}()

// 当配置初始化完成后，可以通过 atomic.Value 的 Load 方法获得最新的配置对象
cfg.Load().(*yamlFile)
```


##### 七彩石 table 使用方法

配置文件需要注明 type 为 table
```yaml
  config:
    rainbow: # 七彩石配置中心
      providers:
       - name: rainbow       # 配置名称
         appid: b4de94fd-fd60-485c-9e99-040ec0f67d08
         group: test_rainbow # 配置所在的 group
         type: table         # 设置为 table
         timeout: 2000
```

具体实现代码参考 [example/rainbow_table.go](https://git.woa.com/trpc-go/trpc-config-rainbow/blob/master/example/rainbow_table.go)

##### 七彩石 group 使用方法

配置文件需要注明 type 为 group
```yaml
  config:
    rainbow: # 七彩石配置中心
      providers:
       - name: rainbow       # 配置名称
         appid: b4de94fd-fd60-485c-9e99-040ec0f67d08
         group: test_rainbow # 配置所在的 group
         type: group         # 设置为 group
         timeout: 2000
```

具体实现代码参考 [example/rainbow_group.go](https://git.woa.com/trpc-go/trpc-config-rainbow/blob/master/example/rainbow_group.go)

##### 如何拉取rainbow_tconf配置
配置文件增加address，env_name设置，env_name从平台连接获取
```yaml
  config:
    rainbow: # 七彩石配置中心
      providers:
       - name: rainbow       # 配置名称
         appid: b4de94fd-fd60-485c-9e99-040ec0f67d08
         group: test_rainbow # 配置所在的 group
         timeout: 2000
         address: http://api.rainbow_tconf.woa.com:8080
         env_name: 03e046b7
```

