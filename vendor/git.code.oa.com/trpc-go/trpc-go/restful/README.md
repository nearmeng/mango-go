# restful 包使用

---

## HttpRule

关于 HttpRule 的详细说明，请查看 trpc-go/internal/httprule 包

下面的几个例子，能直观地展示 HttpRule 到底要怎么使用：

**一、将 URL Path 里面匹配 messages/* 的内容作为 name 字段值：**

```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (google.api.http) = {
        get: "/v1/{name=messages/*}"
    };
  }
}

message GetMessageRequest {
  string name = 1; // Mapped to URL path.
}

message Message {
  string text = 1; // The resource content.
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | GET /v1/messages/123456 | GetMessage(name: "messages/123456") |
 
**二、较为复杂的嵌套 message 构造，URL Path 里的 123456 作为 message_id，sub.subfield 的值作为嵌套 message 里的 subfield：**
 
```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (google.api.http) = {
        get:"/v1/messages/{message_id}"
    };
  }
}

message GetMessageRequest {
  message SubMessage {
    string subfield = 1;
  }
  string message_id = 1; // Mapped to URL path.
  int64 revision = 2;    // Mapped to URL query parameter `revision`.
  SubMessage sub = 3;    // Mapped to URL query parameter `sub.subfield`.
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | GET /v1/messages/123456?revision=2&sub.subfield=foo | GetMessage(message_id: "123456" revision: 2 sub: SubMessage(subfield: "foo")) |
 
**三、将 HTTP Body 的整体作为 Message 类型解析，即将 "Hi!" 作为 message.text 的值：**
 
```protobuf
service Messaging {
  rpc UpdateMessage(UpdateMessageRequest) returns (Message) {
    option (google.api.http) = {
      post: "/v1/messages/{message_id}"
      body: "message"
    };
  }
}

message UpdateMessageRequest {
  string message_id = 1; // mapped to the URL
  Message message = 2;   // mapped to the body
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | POST /v1/messages/123456 { "text": "Hi!" } | UpdateMessage(message_id: "123456" message { text: "Hi!" }) |
 
**四、将 HTTP Body 里的字段解析为 Message 的 text 字段：**
 
```protobuf
service Messaging {
  rpc UpdateMessage(Message) returns (Message) {
    option (google.api.http) = {
      post: "/v1/messages/{message_id}"
      body: "*"
    };
  }
}

message Message {
  string message_id = 1;
  string text = 2;
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | POST/v1/messages/123456 { "text": "Hi!" } | UpdateMessage(message_id: "123456" text: "Hi!") |
 
**五、使用 additional_bindings 表示追加绑定的 API：**
 
```protobuf
service Messaging {
  rpc GetMessage(GetMessageRequest) returns (Message) {
    option (google.api.http) = {
      get: "/v1/messages/{message_id}"
      additional_bindings {
        get: "/v1/users/{user_id}/messages/{message_id}"
      }
    };
  }
}

message GetMessageRequest {
  string message_id = 1;
  string user_id = 2;
}
```

上述 HttpRule 可得以下映射：

 | HTTP | tRPC |
 | ----- | ----- |
 | GET /v1/messages/123456 | GetMessage(message_id: "123456") |
 | GET /v1/users/me/messages/123456 | GetMessage(user_id: "me" message_id: "123456") |

## 搭建 RESTful 服务步骤

理解了 HttpRule 后，我们来看一下具体要如何开启 tRPC-Go 的 RESTful 服务。

**一、PB 定义**

先更新 ```trpc-go-cmdline``` 工具到最新版本，要使用 **trpc.api.http** 注解，需要 import 一个 proto 文件：

```protobuf
import "trpc/api/annotations.proto";
```

我们还是定义一个 Greeter 服务 的 PB:

```protobuf
...

import "trpc/api/annotations.proto";

// Greeter 服务
service Greeter {
  rpc SayHello(HelloRequest) returns (HelloReply) {
    option (trpc.api.http) = {
      post: "/v1/foobar"
      body: "*"
      additional_bindings: {
        post: "/v1/foo/{name}"
      }
    };
  }
}

// Hello 请求
message HelloRequest {
  string name = 1;
  ...
}  
  
...
```

**二、生成桩代码**

直接用 ```trpc create``` 命令生成桩代码。

**三、配置**

和其他协议配置一样，```trpc_go.yaml``` 里面 service 的 protocol 配置成 ```restful``` 即可

```yaml
server: 
  ...
  service:                                         
    - name : trpc.test.helloworld.Greeter      
      ip: 127.0.0.1                            
      #nic: eth0
      port: 8080                
      network: tcp                             
      protocol: restful              
      timeout: 1000
```

更普遍的场景是，我们会配置一个 tRPC 协议的 service，再加一个 RESTful 协议的 service，这样就能做到一套 PB 文件同时支持提供 RPC 服务和 RESTful 服务：

```yaml
server: 
  ...
  service:                                         
    - name : trpc.test.helloworld.Greeter1      
      ip: 127.0.0.1                            
      #nic: eth0
      port: 12345                
      network: tcp                             
      protocol: trpc              
      timeout: 1000
    - name : trpc.test.helloworld.Greeter2      
      ip: 127.0.0.1                            
      #nic: eth0
      port: 54321                
      network: tcp                             
      protocol: restful              
      timeout: 1000
```

***注意：tRPC 每个 service 必须配置不同的端口。***

**四、启动服务**

启动服务和其他协议方式一致：

```go
package main

import (
    ...

    pb "git.code.oa.com/trpc-go/trpc-go/examples/restful/helloworld"
)

func main() {
    
    s := trpc.NewServer()
    
    pb.RegisterGreeterService(s, &greeterServerImpl{})
    
    // 启动
    if err := s.Serve(); err != nil {
    	...
    }
}
```

**五、调用**

搭建的是 RESTful 服务，所以请用任意的 REST 客户端调用，不支持用 NewXXXClientProxy 的 RPC 方式调用：

```go
package main

import "net/http"

func main() {

    ...

    // 原生 HTTP 调用
    req, err := http.NewRequest("POST", "http://127.0.0.1:8080/v1/foobar", bytes.Newbuffer([]byte(`{"name": "xyz"}`)))
    if err != nil {
        ...
    }
    
    cli := http.Client{}
    resp, err := cli.Do(req)
    if err != nil {
        ...
    }
    
    ...
  
}  
    
```

当然如果上面第三点【配置】中，如果配置了 tRPC 协议的 service，我们还是可以通过 NewXXXClientProxy 的 RPC 方式去调用 tRPC 协议的 service，注意区分端口。

**六、自定义 HTTP 头到 RPC Context 映射**

HttpRule 解决的是 tRPC Message Body 和 HTTP/JSON 之间的转码，那么 HTTP 请求如何传递 RPC 调用的上下文呢？这就需要定义 HTTP 头到 RPC Context 映射。

RESTful 服务的 HeaderMatcher 定义如下：

```go
type HeaderMatcher func(
    ctx context.Context,
    w http.ResponseWriter,
    req *http.Request,
    serviceName, methodName string,
) (context.Context, error) {
```

默认的 HeaderMatcher 处理如下：

```go
var defaultHeaderMatcher = func(
    ctx context.Context,
    w http.ResponseWriter,
    req *http.Request,
    serviceName, methodName string,
) (context.Context, error) {
    // 注意：必须要往 ctx 里面塞 codec.Msg，并且指定目标 service 和 method 名
    // 下面这段代码是必须的
    ctx, msg := codec.WithNewMessage(ctx)
    msg.WithClientRPCName(methodName)
    msg.WithCalleeServiceName(serviceName)
    msg.WithSerializationType(codec.SerializationTypePB)
    
    return ctx, nil
}
```

用户可以通过 ```WithOptions``` 的方式设置 HeaderMatcher：

```go
service := server.New(server.WithRESTOptions(restful.WithHeaderMatcher(xxx)))
```

**七、自定义回包处理 [设置请求处理成功的返回码]**

HttpRule 的 response_body 字段指定了 RPC 响应，譬如上面例子中的 HelloReply 要整个或者将其某个字段序列化到 HTTP Response Body 里面。但是用户可能想额外做一些自定义的操作，譬如：设置成功时候的响应码。

RESTful 服务的自定义回包处理函数定义如下：

```go
type CustomResponseHandler func(
    ctx context.Context,
    w http.ResponseWriter,
    r *http.Request,
    resp proto.Message,
    body []byte,
) error
```

trpc-go/restful 包提供了一个让用户设置请求处理成功时候的响应码的函数：

```go
func SetStatusCodeOnSucceed(ctx context.Context, code int) {}
```

默认的自定义回包处理函数如下：

```go
var defaultResponseHandler = func(
    ctx context.Context,
    w http.ResponseWriter,
    r *http.Request,
    resp proto.Message,
    body []byte,
) error {
    // 压缩
    var writer io.Writer = w
    _, compressor := compressorForRequest(r)
    if compressor != nil {
        writeCloser, err := compressor.Compress(w)
        if err != nil {
            return fmt.Errorf("failed to compress resp body: %w", err)
        }
        defer writeCloser.Close()
        w.Header().Set(headerContentEncoding, compressor.ContentEncoding())
        writer = writeCloser
    }

    // 设置响应码
    statusCode := GetStatusCodeOnSucceed(ctx)
    w.WriteHeader(statusCode)

    // 设置 body
    if statusCode != http.StatusNoContent && statusCode != http.StatusNotModified {
        writer.Write(body)
    }

    return nil
}
```

如果使用默认自定义回包处理函数，则支持用户在自己的 RPC 处理函数中设置返回码（不设置则成功返回 200）：

```go
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) (err error) {   
    ...
    
    restful.SetStatusCodeOnSucceed(ctx, 200) // 设置成功时返回码
    return nil
}
```

用户可以通过 ```WithOptions``` 的方式定义回包处理：

```go
var xxxResponseHandler = func(
    ctx context.Context,
    w http.ResponseWriter,
    r *http.Request,
    resp proto.Message,
    body []byte,
) error {
    reply, ok := resp.(*pb.HelloReply)
    if !ok {
        return errors.New("xxx")
    }
    
    ...
    
    w.Header().Set("x", "y")
    expiration := time.Now()
    expiration := expiration.AddDate(1, 0, 0)
    cookie := http.Cookie{Name: "abc", Value: "def", Expires: expiration}
    http.SetCookie(w, &cookie)
    
    w.Write(body)
    
    return nil
}

...
   
service := server.New(server.WithRESTOptions(restful.WithResponseHandler(xxxResponseHandler)))
```

**八、自定义错误处理 [错误码]**

RESTful 错误处理函数定义如下：

```go
type ErrorHandler func(context.Context, http.ResponseWriter, *http.Request, error)
```

用户可以通过 ```WithOptions``` 的方式定义错误处理：

```go
var xxxErrorHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
    if err == errors.New("say hello failed") {
        w.WriteHeader(500)
    }

    ...

}

service := server.New(server.WithRESTOptions(restful.WithErrorHandler(xxxErrorHandler)))
```

***建议使用 trpc-go/restful 包默认的错误处理函数，或者参考实现用户自己的错误处理函数。***

关于**错误码：**

如果 RPC 处理过程中返回了 trpc-go/errs 包定义的错误类型，trpc-go/restful 默认的错误处理函数会将 tRPC 的错误码都映射为 HTTP 错误码。如果用户想自己决定返回的某个错误用什么错误码，请使用 trpc-go/restful 包定义的 ```WithStatusCode``` :

```go
type WithStatusCode struct {
	StatusCode int
	Err        error
}
```

将自己的 error 包起来并返回，如：

```go
func (s *greeterServerImpl) SayHello(ctx context.Context, req *hpb.HelloRequest, rsp *hpb.HelloReply) (err error) {
    if req.Name != "xyz" {
        return &restful.WithStatusCode{
            StatusCode: 400,
            Err:        errors.New("test error"),
        }
    }
    return nil
}
```

如果错误类型不是 trpc-go/errs 的 Error 类型，也没用 trpc-go/restful 包定义的 ```WithStatusCode``` 包起来，则默认错误码返回 500 。

**九、Body 序列化与压缩**

和普通 REST 请求一样，通过 HTTP 头指定，支持比较主流的几个。

>  **序列化支持的 Content-Type (或 Accept)： application/json，application/x-www-form-urlencoded，application/octet-stream。默认为 application/json。** 


序列化接口定义如下：

```go
type Serializer interface {
    // Marshal 把 tRPC message 或其中一个字段序列化到 http body
    Marshal(v interface{}) ([]byte, error)
    // Unmarshal 把 http body 反序列化到 tRPC message 或其中一个字段
    Unmarshal(data []byte, v interface{}) error
    // Name Serializer 名字
    Name() string
    // ContentType http 回包时设置的 Content-Type
    ContentType() string
}
```

**用户可自己实现并通过 ```restful.RegisterSerializer()``` 函数注册。**

> **压缩支持 Content-Encoding (或 Accept-Encoding): gzip。默认不压缩。**

压缩接口定义如下：

```go
type Compressor interface {
    // Compress 压缩
    Compress(w io.Writer) (io.WriteCloser, error)
    // Decompress 解压缩
    Decompress(r io.Reader) (io.Reader, error)
    // Name 表示 Compressor 名字
    Name() string
    // ContentEncoding 表示 http 回包时设置的 Content-Encoding
    ContentEncoding() string
}
```

**用户可自己实现并通过 ```restful.RegisterCompressor()``` 函数注册。**
