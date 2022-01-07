# tRPC-Go http协议
tRPC-Go框架默认支持http协议，与传统的web server不一样，trpc是以rpc的方式提供http服务，流程一般是先定义proto协议文件，再生成http server
http协议与trpc协议保持一致，当返回失败时，body为空，错误码错误信息放在http header

## 自由切换http/trpc协议，只需要在配置文件修改protocol字段即可

```yaml
server:                                            #服务端配置
  app: test                                        #业务的应用名
  server: Greeter                                  #进程服务名
  service:                                         #业务服务提供的service，可以有多个
    - name: trpc.test.helloworld.Greeter           #service的路由名称
      ip: 127.0.0.1                                #服务监听ip地址 可使用占位符 ${ip},ip和nic二选一，优先ip
      port: 8000                                   #服务监听端口 可使用占位符 ${port}
      protocol: http                               #应用层协议 trpc http
```

## 自定义url path，url默认为 /package.service/method，可通过alias参数自定义任意url 
- 定义协议：
```protobuf
syntax = "proto3";
package trpc.app.server;
option go_package="git.code.oa.com/trpcprotocol/app/server";

import "trpc.proto";

message Request {
	bytes req = 1;
}

message Reply {
	bytes rsp = 1;
}

service Greeter {
	rpc SayHello(Request) returns (Reply) {
		option (trpc.alias) = "/cgi-bin/module/say_hello";
	};
}

```
- 生成代码：
```
trpc create --protofile=a.proto --protocol=http
```

## 自定义错误码处理函数
默认错误码处理函数会将错误码填充到http header的 trpc-ret/trpc-func-ret 字段中，也可以通过自己定义ErrorHandler来替换

```golang
import (
	"net/http"

	"git.code.oa.com/trpc-go/trpc-go/errs"

	thttp "git.code.oa.com/trpc-go/trpc-go/http"
	trpc "git.code.oa.com/trpc-go/trpc-go"
)

func init() {

	thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
		// 一般自己定义retcode retmsg字段，并组成json写到response body里面
		w.Write([]byte(fmt.Sprintf(`{"retcode":%d, "retmsg":"%s"}`, e.Code, e.Msg)))
		// 每个业务团队可以定义到自己的git上，业务代码import进来即可
	}
}

```

## 无协议文件自定义路由, 常用于重构兼容老服务和文件传输

### 注册无协议文件服务

#### 全局只支持无协议文件http服务(谨慎使用)

> 这种注册方式,无需修改配置文件,不能和有协议文件服务同时存在,谨慎使用

```go
func main() {

	s := trpc.NewServer()

	thttp.HandleFunc("/xxx/xxx", handle) // handle函数见下一节
	thttp.RegisterDefaultService(s) // 注意：这种注册方式,无协议文件服务不能和有协议文件服务同时存在

	s.Serve()
}
```

#### 同时支持无协议文件服务和有协议文件服务

> 注意：无协议文件服务和有协议文件服务同时存在,必须在配置文件中配置service,并使用s.Service()加载对应service

在配置文件中配置service,协议为`http_no_protocol`,http2为`http2_no_protocol`

```yaml
server:
    service: #业务服务提供的service，可以有多个
      - name: trpc.xxx #service的路由名称
        network: tcp #网络监听类型  tcp udp
        protocol: http_no_protocol #应用层协议 http_no_protocol
        timeout: 1000 #请求最长处理时间 单位 毫秒
        # ip: xxx
        # port: xxx #服务监听端口 可使用占位符 ${port}
```

代码

```golang
import (
	"net/http"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/log"

	thttp "git.code.oa.com/trpc-go/trpc-go/http"
	trpc "git.code.oa.com/trpc-go/trpc-go"
)

func main() {

	s := trpc.NewServer()

	thttp.HandleFunc("/xxx", handle) // handle函数见下一节
    // 注意：无协议文件服务和有协议文件服务同时存在,必须在配置文件中配置service,并使用s.Service()加载对应service
    thttp.RegisterNoProtocolService(s.Service("trpc.xxx")) 

	s.Serve()
}


```



### handle函数

```go
func handle(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()
	log.DebugContextf(ctx, "xxx")

	// 请求后端，一般不需要自己设置reqhead rsphead，特殊需求如携带header时才需要
	// 设置自定义cookie: 
	r.AddCookie(&http.Cookie{
		Name:    "clientcookieid",
		Value:   "id",
		Expires: time.Now().Add(time.Hour),
	})
	reqhead := &thttp.ClientReqHeader{
		Host:   "www.qq.com", // host只能通过这里设置
		Method: r.Method, // 可以自定义 Get Post
		Header: r.Header, // 有已存在的Header可以直接赋值，没有的话，也可以直接通过reqhead.AddHeader逐个添加
	}
	rsphead := &thttp.ClientRspHeader{}
	
	// 后端服务，自己搭建
	req := &pb.HelloRequest{Msg: "hello"}
	proxy := pb.NewGreeterClientProxy()
	rsp, err := proxy.SayHello(ctx, req,
		client.WithProtocol("http"), // 后端服务的协议
		client.WithReqHead(reqhead),
		client.WithRspHead(rsphead), // ClientReqHeader 的设置需要在每次调用时指定
		client.WithSerializationType(codec.SerializationTypePB),
	)
	if err != nil {
		return errs.New(1000, "http sayhello request fail")
	}

	// 取出回包cookie：rsphead.Response.Cookies()
	
	// 注意!! 使用ResponseWriter回包时，Set/WriteHeader/Write这三个方法必须严格按照以下顺序调用：
	// w.Header().Set("Content-type", "application/text")
	// w.WriteHeader(403)
	// w.Write([]byte("response body"))

	return nil
}
```



## 无协议文件调用下游http服务

```golang
func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {

	// 自己指定下游httpserver的服务名，自己随便定义，主要用于监控上报
	// 携带http header请求下游，请参考上面的例子，在NewClientProxy添加reqhead即可
	proxy := thttp.NewClientProxy("trpc.http.server.service")

	// req rsp 自己定义请求/响应结构体，并自己指定序列化方式，不填则默认为json，框架内部会自动序列化
	// 如果发送的是 form 请求(content-type为application/x-www-form-urlencoded)，则 req 必须为 url.Values，并且设置了序列化方式为codec.SerializationTypeForm
	// rsp会根据下游返回的content-type，自动选择序列化方式，如果下游服务content-type不符规范，可以自己调用SetContentType兼容
	err := proxy.Post(ctx, "/cgi-bin/add", req, rsp)
	if err != nil {
		return err
	}

	return nil
}
```

```yaml
client:                                            #客户端调用的后端配置
  timeout: 1000                                    #针对所有后端的请求最长处理时间
  namespace: Development                           #针对所有后端的环境
  service:                                         #针对单个后端的配置
    - name: trpc.http.server.service               #下游http服务的service name 
      target: ip://127.0.0.1:8000                  #下游http服务的请求地址
      serialization: 0                             #上游使用get方式请求时，必须设置下游的序列化方式
```
