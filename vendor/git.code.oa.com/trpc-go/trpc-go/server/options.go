package server

import (
	"context"
	"net"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/overloadctrl"
	"git.code.oa.com/trpc-go/trpc-go/restful"
	"git.code.oa.com/trpc-go/trpc-go/transport"
)

// Options 服务端调用参数
type Options struct {
	Namespace   string // 当前服务命名空间 正式环境 Production 测试环境 Development
	EnvName     string // 当前环境
	SetName     string // set分组
	ServiceName string // 当前服务的 service name

	Address                  string        // 监听地址 ip:port
	Timeout                  time.Duration // 请求最长处理时间
	DisableRequestTimeout    bool          // 禁用继承上游的超时时间
	CurrentSerializationType int
	CurrentCompressType      int

	protocol   string // 业务协议 trpc http ...
	network    string // 网络类型 tcp udp ...
	handlerSet bool   // 用户是否自己定义handler

	ServeOptions []transport.ListenServeOption
	Transport    transport.ServerTransport

	OverloadCtrl overloadctrl.OverloadController

	Registry        registry.Registry
	Codec           codec.Codec
	Filters         filter.Chain                    // 链式拦截器
	StreamHandle    StreamHandle                    // 流式的包处理器
	StreamTransport transport.ServerStreamTransport // 服务端流式Transport
	MaxWindowSize   uint32                          // 设置服务端流式接收端最大的window大小
	CloseWaitTime   time.Duration                   // 关闭服务时,反注册后到真正停止服务之间的等待时间,来支持无损更新

	RESTOptions []restful.Option // RESTful 路由选项
}

// StreamHandle 流式包的处理器
type StreamHandle interface {
	// StreamHandleFunc 数据处理函数
	StreamHandleFunc(ctx context.Context, sh StreamHandler, req []byte) ([]byte, error)
	// Init 初始化，主要是传入和保存Options
	Init(opts *Options) error
}

// Option 服务启动参数工具函数
type Option func(*Options)

// WithNamespace 设置 namespace
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

// WithStreamTransport 设置流式Transport
func WithStreamTransport(st transport.ServerStreamTransport) Option {
	return func(o *Options) {
		o.StreamTransport = st
	}
}

// WithEnvName 设置当前环境
func WithEnvName(envName string) Option {
	return func(o *Options) {
		o.EnvName = envName
	}
}

// WithSetName 设置set分组
func WithSetName(setName string) Option {
	return func(o *Options) {
		o.SetName = setName
	}
}

// WithServiceName 当前服务service name
func WithServiceName(s string) Option {
	return func(o *Options) {
		o.ServiceName = s
		o.ServeOptions = append(o.ServeOptions, transport.WithServiceName(s))
	}
}

// WithFilter 添加服务端拦截器，支持在 业务handler处理函数前后 拦截处理
func WithFilter(fs filter.Filter) Option {
	return func(o *Options) {
		o.Filters = append(o.Filters, fs)
	}
}

// WithFilters 添加服务端拦截器，支持在 业务handler处理函数前后 拦截处理
func WithFilters(fs []filter.Filter) Option {
	return func(o *Options) {
		o.Filters = append(o.Filters, fs...)
	}
}

// WithAddress 指定server监听地址 ip:port or :port
func WithAddress(s string) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithListenAddress(s))
		o.Address = s
	}
}

// WithTLS 指定server tls文件地址, certFile服务端自身证书，keyFile服务端自身秘钥。
// caFile CA证书，用于开启双向认证，校验client证书，以更严格识别客户端的身份，限制客户端访问，
// server一般是单向认证，为空不校验，也可以传入caFile="root"表示使用本机安装的ca证书来验证client。
// 证书标准当前只支持X.509。
func WithTLS(certFile, keyFile, caFile string) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithServeTLS(certFile, keyFile, caFile))
	}
}

// WithNetwork 指定server监听网络 tcp or udp 默认tcp
func WithNetwork(s string) Option {
	return func(o *Options) {
		o.network = s
		o.ServeOptions = append(o.ServeOptions, transport.WithListenNetwork(s))
	}
}

// WithListener 允许用户自己设置listener，用于自己操作accept read/write等特殊逻辑
func WithListener(lis net.Listener) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithListener(lis))
	}
}

// WithServerAsync 设置服务端异步处理
// 其他框架调用trpc，比如TAF调用的时候会使用长连接，这个时候TRPC服务端不能并发处理，导致超时
// 详见issue https://git.code.oa.com/trpc-go/trpc-go/issues/113
func WithServerAsync(serverAsync bool) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithServerAsync(serverAsync))
	}
}

// WithWritev 设置服务端批量发包(Writev系统调用)
func WithWritev(writev bool) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithWritev(writev))
	}
}

// WithMaxRoutines 设置服务端异步处理最大协程数限制
// 建议设置可以处理预期QPS的协程数的2倍大小，不低于MAXPROCS
// 不设置或者设置为0，默认值为(1<<31 - 1)
// 协程数限制只有在启用异步处理的时候才会生效，如果使用同步模式不生效
func WithMaxRoutines(routines int) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithMaxRoutines(routines))
	}
}

// WithTimeout 指定单个请求整体超时时间 默认没有超时
func WithTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// WithDisableRequestTimeout 禁用继承上游超时时间
func WithDisableRequestTimeout(disable bool) Option {
	return func(o *Options) {
		o.DisableRequestTimeout = disable
	}
}

// WithRegistry 指定server服务注册中心, 一个服务只能支持一个registry
func WithRegistry(r registry.Registry) Option {
	return func(o *Options) {
		o.Registry = r
	}
}

// WithTransport 替换底层server通信层
func WithTransport(t transport.ServerTransport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// WithProtocol 指定服务协议名字 如 trpc 内部设置 framerbuilder codec
func WithProtocol(s string) Option {
	return func(o *Options) {
		o.protocol = s
		o.Codec = codec.GetServer(s)
		fb := transport.GetFramerBuilder(s)
		if fb != nil {
			o.ServeOptions = append(o.ServeOptions, transport.WithServerFramerBuilder(fb))
		}
		trans := transport.GetServerTransport(s)
		if trans != nil {
			o.Transport = trans
		}
	}
}

// WithHandler 指定server transport处理函数，默认是server本身，内部自动调用codec 也可以自己替换
func WithHandler(h transport.Handler) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithHandler(h))
		o.handlerSet = true
	}
}

// WithCurrentSerializationType 设置当前请求序列化方式
// 常用于代理透传不序列化场景, 默认序列化方式从协议字段里面获取，当设置此值时，以该值为准
func WithCurrentSerializationType(t int) Option {
	return func(o *Options) {
		o.CurrentSerializationType = t
	}
}

// WithCurrentCompressType 设置当前请求解压缩方式
func WithCurrentCompressType(t int) Option {
	return func(o *Options) {
		o.CurrentCompressType = t
	}
}

// WithMaxWindowSize 设置流式最大的接收窗口
func WithMaxWindowSize(w uint32) Option {
	return func(o *Options) {
		o.MaxWindowSize = w
	}
}

// WithCloseWaitTime 关闭服务时,反注册后到真正停止服务之间的等待时间,来支持无损更新,单位ms, 默认0,最大10s
func WithCloseWaitTime(t time.Duration) Option {
	return func(o *Options) {
		o.CloseWaitTime = t
	}
}

// WithRESTOptions 设置 RESTful 路由选项
func WithRESTOptions(opts ...restful.Option) Option {
	return func(o *Options) {
		o.RESTOptions = append([]restful.Option{}, opts...)
	}
}

// WithOverloadCtrl 设置过载保护策略。
func WithOverloadCtrl(oc overloadctrl.OverloadController) Option {
	return func(o *Options) {
		o.OverloadCtrl = oc
	}
}

// WithIdleTimeout 设置服务端连接空闲超时
func WithIdleTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.ServeOptions = append(o.ServeOptions, transport.WithServerIdleTimeout(t))
	}
}
