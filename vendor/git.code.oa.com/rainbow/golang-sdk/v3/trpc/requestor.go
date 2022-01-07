package trpc

import (
	"context"
	"fmt"

	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/version"
	v3 "git.code.oa.com/rainbow/proto/api/configv3"
	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/selector"

	// register
	_ "git.code.oa.com/trpc-go/trpc-selector-cl5"
)

// Requestor 请求
type Requestor struct {
	types.AddressBase
	proxyPull    v3.ConfigsClientProxy
	proxyPolling v3.ConfigsClientProxy
	sdkopts      types.InitOptions // 本地sdk参数
	trpcopts     []client.Option   // 调用trpc参数
	sgnp         *types.SignatureParams
}

// Init 初始化
// TODO 参数校验
func (r *Requestor) Init(o types.InitOptions) (err error) {
	err = r.ParseAddress(o.ConnectStr)
	if err != nil {
		return err
	}
	r.sdkopts = o
	opts := []client.Option{
		client.WithServiceName("trpc.config_service.config_service"),
		client.WithProtocol("trpc"),
		client.WithNetwork("tcp"),
		client.WithTarget(r.sdkopts.ConnectStr),
		client.WithTimeout(r.sdkopts.TimeoutCS),
		client.WithMetaData(version.Header, []byte(version.Version)),
	}
	if r.Type == "polaris" {
		opts = []client.Option{
			client.WithServiceName(r.Addr),
			client.WithNamespace("Production"),
			client.WithProtocol("trpc"),
			client.WithNetwork("tcp"),
			client.WithTarget(r.sdkopts.ConnectStr),
			client.WithTimeout(r.sdkopts.TimeoutCS),
			client.WithMetaData(version.Header, []byte(version.Version)),
		}
		selector.RegisterDefault()
	}

	r.sgnp = types.DefaultNew(r.sdkopts.AppID, r.sdkopts.UserID, r.sdkopts.HmacWay)
	err = r.sgnp.SignedString([]byte(r.sdkopts.UserKey), nil)
	if err != nil {
		return fmt.Errorf("SignedString Error=%s", err.Error())
	}
	r.trpcopts = opts
	r.proxyPull = v3.NewConfigsClientProxy(opts...)
	r.proxyPolling = v3.NewConfigsClientProxy(opts...)
	return nil
}

func (r *Requestor) fillOpts(getOpts *types.GetOptions) ([]client.Option, error) {
	var sgnp *types.SignatureParams
	userKey := r.sdkopts.UserKey

	// 此次请求带有签名
	if getOpts.UserID != "" && getOpts.UserKey != "" {
		sgnp = types.DefaultNew(getOpts.AppID, getOpts.UserID, getOpts.HmacWay)
		userKey = getOpts.UserKey
	} else {
		// 没带用全局签名
		sgnp = types.DefaultNew(r.sdkopts.AppID, r.sdkopts.UserID, r.sdkopts.HmacWay)
	}
	err := sgnp.SignedString([]byte(userKey), nil)
	if err != nil {
		return nil, fmt.Errorf("SignedString Error=%s", err.Error())
	}
	opts := make([]client.Option, 0, 10)
	// 打开签名校验
	if r.sdkopts.OpenSign {
		opts = append(opts,
			client.WithMetaData("rainbow_sgn_type", []byte(sgnp.SgnType)),
			client.WithMetaData("rainbow_version", []byte(sgnp.Version)),
			client.WithMetaData("rainbow_app_id", []byte(sgnp.AppID)),
			client.WithMetaData("rainbow_user_id", []byte(sgnp.UserID)),
			client.WithMetaData("rainbow_timestamp", []byte(sgnp.Timestamp)),
			client.WithMetaData("rainbow_nonce", []byte(sgnp.Nonce)),
			client.WithMetaData("rainbow_sgn_method", []byte(sgnp.SgnMethod)),
			client.WithMetaData("rainbow_sgn_body", []byte(sgnp.SgnBody)),
			client.WithMetaData("rainbow_signature", []byte(sgnp.Signature)),
		)
	}
	return opts, nil
}

// Getdatas 拉取配置
func (r *Requestor) Getdatas(ctx context.Context, req *v3.ReqGetDatas,
	getOpts *types.GetOptions) (rsp *v3.RspGetDatas, err error) {
	opts, err := r.fillOpts(getOpts)
	if err != nil {
		return nil, err
	}
	rsp, err = r.proxyPull.Getdatas(ctx, req, opts...)
	return rsp, err
}

// Poll 长轮询
func (r *Requestor) Poll(ctx context.Context, req *v3.ReqPoll,
	getOpts *types.GetOptions) (rsp *v3.RspPoll, err error) {
	opts, err := r.fillOpts(getOpts)
	if err != nil {
		return nil, err
	}
	opt := client.WithTimeout(r.sdkopts.TimeoutPolling)
	opts = append(opts, opt)
	rsp, err = r.proxyPolling.Poll(ctx, req, opts...)
	return rsp, err
}

// Heartbeat 心跳接口，记录客户端类型和版本
/*func (r *Requestor) Heartbeat(ctx context.Context, req *config.ReqHeartbeat,
	getOpts *types.GetOptions) (rsp *config.RspHeartbeat, err error) {
	opts, err := r.fillOpts(getOpts)
	if err != nil {
		return nil, err
	}
	rsp, err = r.proxyPull.Heartbeat(ctx, req, opts...)
	return rsp, err
}*/
