package transport

import (
	"net"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
)

// ListenServeOptions server每次启动参数
type ListenServeOptions struct {
	ServiceName   string
	Address       string
	Network       string
	Handler       Handler
	FramerBuilder codec.FramerBuilder
	Listener      net.Listener

	CACertFile  string        // ca证书
	TLSCertFile string        // server证书
	TLSKeyFile  string        // server秘钥
	Routines    int           // 协程池协程数量
	ServerAsync bool          // 服务端启用异步处理
	Writev      bool          // 服务端启用批量发送
	CopyFrame   bool          // 是否拷贝Frame到后端
	IdleTimeout time.Duration // 服务端连接空闲超时
}

// ListenServeOption function type for config listenServeOptions
type ListenServeOption func(*ListenServeOptions)

// WithServiceName 设置 service name
func WithServiceName(name string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.ServiceName = name
	}
}

// WithServerFramerBuilder 设置FramerBuilder
func WithServerFramerBuilder(fb codec.FramerBuilder) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.FramerBuilder = fb
	}
}

// WithListenAddress 设置ListenAddress
func WithListenAddress(address string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Address = address
	}
}

// WithListenNetwork 设置ListenNetwork
func WithListenNetwork(network string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Network = network
	}
}

// WithListener 允许用户自己设置listener，用于自己操作accept read/write等特殊逻辑
func WithListener(lis net.Listener) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Listener = lis
	}
}

// WithHandler 设置业务处理抽象接口Handler
func WithHandler(handler Handler) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Handler = handler
	}
}

// WithServeTLS 设置服务支持TLS
func WithServeTLS(certFile, keyFile, caFile string) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.TLSCertFile = certFile
		opts.TLSKeyFile = keyFile
		opts.CACertFile = caFile
	}
}

// WithServerAsync 设置服务端异步处理
// 其他框架调用trpc，调用的时候可能会使用长连接，这个时候TRPC服务端不能并发处理，导致超时
// 该从监听选项一直传递到每个TCP连接
func WithServerAsync(serverAsync bool) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.ServerAsync = serverAsync
	}
}

// WithWritev 设置服务端启用批量发送(Writev系统调用)
func WithWritev(writev bool) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Writev = writev
	}
}

// WithMaxRoutines 设置服务端异步处理最大协程数限制
// 建议设置可以处理预期QPS的协程数的2倍大小，不低于MAXPROCS
// 不设置或者设置为0，默认值为(1<<31 - 1)
// 协程数限制只有在启用异步处理的时候才会生效，如果使用同步模式不生效
func WithMaxRoutines(routines int) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.Routines = routines
	}
}

// WithCopyFrame 设置是否拷贝整个frame到后端
// 流式传输场景下，使用服务端使用同步模型，但由于流式后端处理是异步的，也需要拷贝整个frame到后端
// 防止被覆盖
func WithCopyFrame(copyFrame bool) ListenServeOption {
	return func(opts *ListenServeOptions) {
		opts.CopyFrame = copyFrame
	}
}

// WithIdleTimeout 设置Server端连接空闲存在时间
func WithServerIdleTimeout(timeout time.Duration) ListenServeOption {
	return func(options *ListenServeOptions) {
		options.IdleTimeout = timeout
	}
}
