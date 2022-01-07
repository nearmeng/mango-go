// Package connpool 连接池
package connpool

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
)

// GetOptions get conn configuration
type GetOptions struct {
	FramerBuilder codec.FramerBuilder
	Ctx           context.Context

	CACertFile    string // ca证书
	TLSCertFile   string // client证书
	TLSKeyFile    string // client秘钥
	TLSServerName string // client校验server的服务名, 不填时默认为http的hostname

	LocalAddr   string        // 建立连接时本地地址，默认随机选择。
	DialTimeout time.Duration // 建立连接超时时间
}

func (opts *GetOptions) getDialCtx(dialTimeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := opts.Ctx
	defer func() {
		// opts.Ctx仅用于传递ctx参数，ctx不建议被数据结构持有
		opts.Ctx = nil
	}()

	for {
		// 如果RPC请求没有设置ctx，则创建新的ctx
		if ctx == nil {
			break
		}
		// 如果RPC请求没有设置ctx超时，则创建新的ctx
		deadline, ok := ctx.Deadline()
		if !ok {
			break
		}
		// 如果RPC请求超时大于设置的超时时间，则创建新的ctx
		d := time.Until(deadline)
		if opts.DialTimeout > 0 && opts.DialTimeout < d {
			break
		}
		return ctx, nil
	}

	if opts.DialTimeout > 0 {
		dialTimeout = opts.DialTimeout
	}
	if dialTimeout == 0 {
		dialTimeout = defaultDialTimeout
	}
	return context.WithTimeout(context.Background(), dialTimeout)
}

// NewGetOptions 创建并初始化GetOptions
func NewGetOptions() GetOptions {
	return GetOptions{}
}

// WithFramerBuilder 设置 FramerBuilder
func (o *GetOptions) WithFramerBuilder(fb codec.FramerBuilder) {
	o.FramerBuilder = fb
}

// WithDialTLS 设置client支持TLS
func (o *GetOptions) WithDialTLS(certFile, keyFile, caFile, serverName string) {
	o.TLSCertFile = certFile
	o.TLSKeyFile = keyFile
	o.CACertFile = caFile
	o.TLSServerName = serverName
}

// WithContext 设置请求的 ctx
func (o *GetOptions) WithContext(ctx context.Context) {
	o.Ctx = ctx
}

// WithLocalAddr 建立连接时指定本地地址，多网卡时默认随机选择
func (o *GetOptions) WithLocalAddr(addr string) {
	o.LocalAddr = addr
}

// WithDialTimeout 建立连接超时时间
func (o *GetOptions) WithDialTimeout(dur time.Duration) {
	o.DialTimeout = dur
}

// GetOption Options helper
// Deprecated: please use PoolWithOptions instead.
type GetOption func(*GetOptions)

// WithFramerBuilder 设置 FramerBuilder
// Deprecated: please use PoolWithOptions instead.
func WithFramerBuilder(fb codec.FramerBuilder) GetOption {
	return func(opts *GetOptions) {
		opts.FramerBuilder = fb
	}
}

// WithDialTLS 设置client支持TLS
// Deprecated: please use PoolWithOptions instead.
func WithDialTLS(certFile, keyFile, caFile, serverName string) GetOption {
	return func(opts *GetOptions) {
		opts.TLSCertFile = certFile
		opts.TLSKeyFile = keyFile
		opts.CACertFile = caFile
		opts.TLSServerName = serverName
	}
}

// WithContext 设置请求的 ctx
// Deprecated: please use PoolWithOptions instead.
func WithContext(ctx context.Context) GetOption {
	return func(opts *GetOptions) {
		opts.Ctx = ctx
	}
}

// Pool client connection pool
type Pool interface {
	Get(network string, address string, timeout time.Duration, opt ...GetOption) (net.Conn, error)
}

// PoolWithOptions 客户端连接池
// PoolWithOptions相比Pool，函数入参直接使用GetOptions数据结构，相比函数选项入参模式，可以减少内存逃逸，提升调用性能
type PoolWithOptions interface {
	GetWithOptions(network string, address string, opt GetOptions) (net.Conn, error)
}

// DialOptions 请求参数
type DialOptions struct {
	Network       string
	Address       string
	LocalAddr     string
	Timeout       time.Duration
	CACertFile    string // ca证书
	TLSCertFile   string // client证书
	TLSKeyFile    string // client秘钥
	TLSServerName string // client校验server的服务名, 不填时默认为http的hostname
}

// Dial 发起请求
func Dial(opts *DialOptions) (net.Conn, error) {
	var localAddr net.Addr
	if opts.LocalAddr != "" {
		var err error
		localAddr, err = net.ResolveTCPAddr(opts.Network, opts.LocalAddr)
		if err != nil {
			return nil, err
		}
	}
	dialer := &net.Dialer{
		Timeout:   opts.Timeout,
		LocalAddr: localAddr,
	}
	if len(opts.CACertFile) == 0 {
		return dialer.Dial(opts.Network, opts.Address)
	}

	tlsConf := &tls.Config{}
	if opts.CACertFile == "none" { // 不需要检验服务证书
		tlsConf.InsecureSkipVerify = true
	} else { // 需要校验服务端证书
		if len(opts.TLSServerName) == 0 {
			opts.TLSServerName = opts.Address
		}
		tlsConf.ServerName = opts.TLSServerName
		certPool, err := getCertPool(opts.CACertFile)
		if err != nil {
			return nil, err
		}

		tlsConf.RootCAs = certPool

		if len(opts.TLSCertFile) != 0 { // https服务开启双向认证，需要传送client自身证书给server
			cert, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSKeyFile)
			if err != nil {
				return nil, errs.NewFrameError(errs.RetClientDecodeFail,
					"client dial load cert file error: "+err.Error())
			}
			tlsConf.Certificates = []tls.Certificate{cert}
		}
	}
	return tls.DialWithDialer(dialer, opts.Network, opts.Address, tlsConf)
}

func getCertPool(caCertFile string) (*x509.CertPool, error) {
	if caCertFile != "root" { // root 表示使用本机安装的root ca证书来验证server，不是root则使用输入ca文件来验证server
		ca, err := ioutil.ReadFile(caCertFile)
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientDecodeFail,
				"client dial read ca file error: "+err.Error())
		}
		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(ca)
		if !ok {
			return nil, errs.NewFrameError(errs.RetClientDecodeFail,
				"client dial AppendCertsFromPEM fail")
		}

		return certPool, nil
	}

	return nil, nil
}
