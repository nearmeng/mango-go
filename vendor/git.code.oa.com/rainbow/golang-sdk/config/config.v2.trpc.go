// Code generated by protoc-gen-trpc. DO NOT EDIT.
// source: config.v2.proto

package config

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
)

import (
	context "context"

	_ "git.code.oa.com/trpc-go/trpc-go"
	client "git.code.oa.com/trpc-go/trpc-go/client"
	codec "git.code.oa.com/trpc-go/trpc-go/codec"
	_ "git.code.oa.com/trpc-go/trpc-go/http"
	server "git.code.oa.com/trpc-go/trpc-go/server"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

/* ************************************ Service Definition ************************************ */
type ConfigService interface {
	//配置拉取
	PullConfigReq(ctx context.Context, req *ReqPullConfig, rsp *RspPullConfig) error
	//配置长轮询
	PollingReq(ctx context.Context, req *ReqPolling, rsp *RspPolling) error
	//配置订阅/增量推送 (未实现)
	SubscribeReq(ctx context.Context, req *ReqSubscribe, rsp *RspSubscribe) error
	// 心跳接口，记录客户端类型和版本
	Heartbeat(ctx context.Context, req *ReqHeartbeat, rsp *RspHeartbeat) error
}

func ConfigService_PullConfigReq_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	req := &ReqPullConfig{}
	rsp := &RspPullConfig{}

	filters, err := f(req)
	if err != nil {
		return nil, err
	}

	handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
		return svr.(ConfigService).PullConfigReq(ctx, reqbody.(*ReqPullConfig), rspbody.(*RspPullConfig))
	}
	err = filters.Handle(ctx, req, rsp, handleFunc)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func ConfigService_PollingReq_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	req := &ReqPolling{}
	rsp := &RspPolling{}

	filters, err := f(req)
	if err != nil {
		return nil, err
	}

	handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
		return svr.(ConfigService).PollingReq(ctx, reqbody.(*ReqPolling), rspbody.(*RspPolling))
	}
	err = filters.Handle(ctx, req, rsp, handleFunc)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func ConfigService_SubscribeReq_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	req := &ReqSubscribe{}
	rsp := &RspSubscribe{}

	filters, err := f(req)
	if err != nil {
		return nil, err
	}

	handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
		return svr.(ConfigService).SubscribeReq(ctx, reqbody.(*ReqSubscribe), rspbody.(*RspSubscribe))
	}
	err = filters.Handle(ctx, req, rsp, handleFunc)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func ConfigService_Heartbeat_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (interface{}, error) {
	req := &ReqHeartbeat{}
	rsp := &RspHeartbeat{}

	filters, err := f(req)
	if err != nil {
		return nil, err
	}

	handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
		return svr.(ConfigService).Heartbeat(ctx, reqbody.(*ReqHeartbeat), rspbody.(*RspHeartbeat))
	}
	err = filters.Handle(ctx, req, rsp, handleFunc)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

// ConfigServiceServiceDesc descriptor for server.RegisterService
var ConfigServiceServiceDesc = server.ServiceDesc{
	ServiceName: "config.v2.ConfigService",
	HandlerType: ((*ConfigService)(nil)),
	Methods: []server.Method{
		{Name: "/config.v2.ConfigService/PullConfigReq", Func: ConfigService_PullConfigReq_Handler},
		{Name: "/config.v2.ConfigService/PollingReq", Func: ConfigService_PollingReq_Handler},
		{Name: "/config.v2.ConfigService/SubscribeReq", Func: ConfigService_SubscribeReq_Handler},
		{Name: "/config.v2.ConfigService/Heartbeat", Func: ConfigService_Heartbeat_Handler},
	},
}

func RegisterConfigService(s server.Service, svr ConfigService) {
	s.Register(&ConfigServiceServiceDesc, svr)
}

/* ************************************ Client Definition ************************************ */
type ConfigServiceClientProxy interface {
	//配置拉取
	PullConfigReq(ctx context.Context, req *ReqPullConfig, opts ...client.Option) (*RspPullConfig, error)
	//配置长轮询
	PollingReq(ctx context.Context, req *ReqPolling, opts ...client.Option) (*RspPolling, error)
	//配置订阅/增量推送 (未实现)
	SubscribeReq(ctx context.Context, req *ReqSubscribe, opts ...client.Option) (*RspSubscribe, error)
	// 心跳接口，记录客户端类型和版本
	Heartbeat(ctx context.Context, req *ReqHeartbeat, opts ...client.Option) (*RspHeartbeat, error)
}

type ConfigServiceClientProxyImpl struct {
	client client.Client
	opts   []client.Option
}

func NewConfigServiceClientProxy(opts ...client.Option) ConfigServiceClientProxy {
	return &ConfigServiceClientProxyImpl{client: client.DefaultClient, opts: opts}
}

//配置拉取
func (c *ConfigServiceClientProxyImpl) PullConfigReq(ctx context.Context, req *ReqPullConfig, opts ...client.Option) (*RspPullConfig, error) {
	ctx, msg := codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(ConfigServiceServiceDesc.Methods[0].Name)
	msg.WithCalleeServiceName(ConfigServiceServiceDesc.ServiceName)

	callopts := make([]client.Option, 0, len(c.opts)+len(opts))
	callopts = append(callopts, c.opts...)
	callopts = append(callopts, opts...)

	rsp := &RspPullConfig{}
	err := c.client.Invoke(ctx, req, rsp, callopts...)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

//配置长轮询
func (c *ConfigServiceClientProxyImpl) PollingReq(ctx context.Context, req *ReqPolling, opts ...client.Option) (*RspPolling, error) {
	ctx, msg := codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(ConfigServiceServiceDesc.Methods[1].Name)
	msg.WithCalleeServiceName(ConfigServiceServiceDesc.ServiceName)

	callopts := make([]client.Option, 0, len(c.opts)+len(opts))
	callopts = append(callopts, c.opts...)
	callopts = append(callopts, opts...)

	rsp := &RspPolling{}
	err := c.client.Invoke(ctx, req, rsp, callopts...)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

//配置订阅/增量推送 (未实现)
func (c *ConfigServiceClientProxyImpl) SubscribeReq(ctx context.Context, req *ReqSubscribe, opts ...client.Option) (*RspSubscribe, error) {
	ctx, msg := codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(ConfigServiceServiceDesc.Methods[2].Name)
	msg.WithCalleeServiceName(ConfigServiceServiceDesc.ServiceName)

	callopts := make([]client.Option, 0, len(c.opts)+len(opts))
	callopts = append(callopts, c.opts...)
	callopts = append(callopts, opts...)

	rsp := &RspSubscribe{}
	err := c.client.Invoke(ctx, req, rsp, callopts...)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

// 心跳接口，记录客户端类型和版本
func (c *ConfigServiceClientProxyImpl) Heartbeat(ctx context.Context, req *ReqHeartbeat, opts ...client.Option) (*RspHeartbeat, error) {
	ctx, msg := codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(ConfigServiceServiceDesc.Methods[3].Name)
	msg.WithCalleeServiceName(ConfigServiceServiceDesc.ServiceName)

	callopts := make([]client.Option, 0, len(c.opts)+len(opts))
	callopts = append(callopts, c.opts...)
	callopts = append(callopts, opts...)

	rsp := &RspHeartbeat{}
	err := c.client.Invoke(ctx, req, rsp, callopts...)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}
