package types

import (
	"context"

	"git.code.oa.com/rainbow/golang-sdk/config"
	v3 "git.code.oa.com/rainbow/proto/api/configv3"
)

// Request 请求接口
type Request interface {
	// Init 初始化
	Init(o InitOptions) (err error)
	// PullConfig 拉取配置
	PullConfig(ctx context.Context, req *config.ReqPullConfig, getOpts *GetOptions) (rsp *config.RspPullConfig, err error)
	// Polling 长轮询
	Polling(ctx context.Context, req *config.ReqPolling, getOpts *GetOptions) (rsp *config.RspPolling, err error)
	// Subscribe 订阅
	Subscribe(ctx context.Context, req *config.ReqSubscribe, getOpts *GetOptions) (rsp *config.RspSubscribe, err error)
	// 心跳接口，记录客户端类型和版本
	Heartbeat(ctx context.Context, req *config.ReqHeartbeat, getOpts *GetOptions) (rsp *config.RspHeartbeat, err error)
}

// RequestV3 请求接口v3
type RequestV3 interface {
	// Init 初始化
	Init(o InitOptions) (err error)
	// Getdatas 获取配置
	Getdatas(ctx context.Context, req *v3.ReqGetDatas, getOpts *GetOptions) (rsp *v3.RspGetDatas, err error)
	// poll 长轮训得到事件，用来监听事件
	// 注意:
	// 由于服务端会hold住请求60秒，所以请确保客户端访问服务端的超时时间要大于60秒
	Poll(ctx context.Context, req *v3.ReqPoll, getOpts *GetOptions) (rsp *v3.RspPoll, err error)
}
