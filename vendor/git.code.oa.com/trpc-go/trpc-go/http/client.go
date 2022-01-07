package http

import (
	"context"
	stdhttp "net/http"
	"strings"

	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-go/codec"
)

// Client http client, 主要用于通用http后端，并非trpc pb协议生成的http server
type Client interface {
	Get(ctx context.Context, path string, rspbody interface{}, opts ...client.Option) error
	Post(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
	Put(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
	Patch(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
	Delete(ctx context.Context, path string, reqbody interface{}, rspbody interface{}, opts ...client.Option) error
}

// httpCli 后端请求结构体
type httpCli struct {
	ServiceName string
	Client      client.Client
	opts        []client.Option
}

// NewClientProxy 新建一个http后端请求代理 必传参数 http服务名: trpc.http.xxx.xxx
// name 后端http服务的服务名，主要用于配置key，监控上报，自己随便定义，格式是: trpc.app.server.service
var NewClientProxy = func(name string, opts ...client.Option) Client {
	c := &httpCli{
		ServiceName: name,
		Client:      client.DefaultClient,
	}
	c.opts = make([]client.Option, 0, len(opts)+1)
	c.opts = append(c.opts, client.WithProtocol("http"))
	c.opts = append(c.opts, opts...)
	return c
}

// Post 使用trpc client发起http post请求
// path url域名后面的字符串: /cgi-bin/addxxx
// reqbody rspbody 请求包 响应包，通过自己指定序列化方式填入具体的类型, 默认使用json
// 用户通过 client.WithClientReqHead 指定了请求头必须保证 httpMethod 为 Post
func (c *httpCli) Post(ctx context.Context, path string, reqbody interface{}, rspbody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	c.setDefaultCallOption(msg, stdhttp.MethodPost, path)
	return c.send(ctx, reqbody, rspbody, opts...)
}

// Put 使用trpc client发起http put请求
// path url域名后面的字符串: /cgi-bin/updatexxx
// reqbody rspbody 请求包 响应包，通过自己指定序列化方式填入具体的类型, 默认使用json
// 用户通过 client.WithClientReqHead 指定了请求头必须保证 httpMethod 为 Put
func (c *httpCli) Put(ctx context.Context, path string, reqbody interface{}, rspbody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	c.setDefaultCallOption(msg, stdhttp.MethodPut, path)
	return c.send(ctx, reqbody, rspbody, opts...)
}

// Patch 使用trpc client发起http patch请求
// path url域名后面的字符串: /cgi-bin/updatexxx
// reqbody rspbody 请求包 响应包，通过自己指定序列化方式填入具体的类型, 默认使用json
// 用户通过 client.WithClientReqHead 指定了请求头必须保证 httpMethod 为 PATCH
func (c *httpCli) Patch(ctx context.Context, path string, reqbody interface{}, rspbody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	c.setDefaultCallOption(msg, stdhttp.MethodPatch, path)
	return c.send(ctx, reqbody, rspbody, opts...)
}

// Delete 使用trpc client发起http delete请求
// path url域名后面的字符串: /cgi-bin/deletexxx
// reqbody rspbody 请求包 响应包，通过自己指定序列化方式填入具体的类型, 默认使用json
// 用户通过 client.WithClientReqHead 指定了请求头必须保证 httpMethod 为 Delete
// delete 可能有body，如果为空，reqbody rspbody 传 nil
func (c *httpCli) Delete(ctx context.Context, path string, reqbody interface{}, rspbody interface{},
	opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	c.setDefaultCallOption(msg, stdhttp.MethodDelete, path)

	return c.send(ctx, reqbody, rspbody, opts...)
}

// Get 使用trpc client发起http get请求
// path url域名后面的字符串: /cgi-bin/getxxx?k1=v1&k2=v2
// rspbody 响应包，通过自己指定序列化方式填入具体的类型, 默认使用json
// 用户通过 client.WithClientReqHead 指定了请求头必须保证 httpMethod 为 Get
func (c *httpCli) Get(ctx context.Context, path string, rspbody interface{}, opts ...client.Option) error {
	ctx, msg := codec.WithCloneMessage(ctx)
	c.setDefaultCallOption(msg, stdhttp.MethodGet, path)
	return c.send(ctx, nil, rspbody, opts...)
}

// send 使用trpc client 发起 http 请求
// path url域名后面的字符串: /cgi-bin/getxxx?k1=v1&k2=v2
func (c *httpCli) send(ctx context.Context, reqbody, rspbody interface{}, opts ...client.Option) error {
	return c.Client.Invoke(ctx, reqbody, rspbody, append(c.opts, opts...)...)
}

// setDefaultCallOption 设置默认调用参数
func (c *httpCli) setDefaultCallOption(msg codec.Msg, method, path string) {
	msg.WithClientRPCName(path)
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(codec.SerializationTypeJSON)

	// 指定httpMethod
	msg.WithClientReqHead(&ClientReqHeader{
		Method: method,
	})
	msg.WithClientRspHead(&ClientRspHeader{})

	// callee method主要用于监控上报方法，有特殊需求可以自己copy这一段代码出去自己修改
	if s := strings.Split(path, "?"); len(s) > 0 {
		msg.WithCalleeMethod(s[0])
	}
}
