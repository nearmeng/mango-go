package transport

import (
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/pool/connpool"
	"git.code.oa.com/trpc-go/trpc-go/pool/multiplexed"
)

// RoundTripOptions 当次请求的可选参数
type RoundTripOptions struct {
	Address               string // IP:Port. 注意：到了transport层，已经过了名字服务解析，所以直接就是IP:Port
	Password              string
	Network               string // tcp/udp
	LocalAddr             string // 建立连接时本地地址，默认随机选择。
	DialTimeout           time.Duration
	Pool                  connpool.Pool // client连接池
	ReqType               RequestType   // SendAndRecv, SendOnly
	FramerBuilder         codec.FramerBuilder
	ConnectionMode        ConnectionMode
	DisableConnectionPool bool // 禁用连接池
	EnableMultiplexed     bool // 开启多路复用
	Multiplexed           *multiplexed.Multiplexed
	Msg                   codec.Msg

	CACertFile    string // ca证书
	TLSCertFile   string // client证书
	TLSKeyFile    string // client秘钥
	TLSServerName string // client校验server的服务名, 不填时默认为http的hostname
}

// ConnectionMode 连接工作模式，Connected或NotConnected
type ConnectionMode bool

// udp 连接模式值
const (
	Connected    = false // udp隔离非相同路径回包
	NotConnected = true  // udp允许非相同路径回包
)

// RequestType 客户端请求类型, 例如 SendAndRecv，SendOnly
type RequestType = codec.RequestType

// 请求类型值
const (
	SendAndRecv RequestType = codec.SendAndRecv //一来一回
	SendOnly    RequestType = codec.SendOnly    //只发不收
)

// RoundTripOption func for setting client RoundTrip
type RoundTripOption func(*RoundTripOptions)

// WithDialAddress set dial address
func WithDialAddress(address string) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Address = address
	}
}

// WithDialPassword set dial password
func WithDialPassword(password string) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Password = password
	}
}

// WithDialNetwork set dial network
func WithDialNetwork(network string) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Network = network
	}
}

// WithDialPool set dial pool
func WithDialPool(pool connpool.Pool) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Pool = pool
	}
}

// WithClientFramerBuilder 设置FramerBuilder
func WithClientFramerBuilder(builder codec.FramerBuilder) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.FramerBuilder = builder
	}
}

// WithReqType 设置请求的类型
func WithReqType(reqType RequestType) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.ReqType = reqType
	}
}

// WithConnectionMode 设置允许的udp回包路径
func WithConnectionMode(connMode ConnectionMode) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.ConnectionMode = connMode
	}
}

// WithDialTLS 设置client支持TLS
func WithDialTLS(certFile, keyFile, caFile, serverName string) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.TLSCertFile = certFile
		opts.TLSKeyFile = keyFile
		opts.CACertFile = caFile
		opts.TLSServerName = serverName
	}
}

// WithDisableConnectionPool 禁用连接池
func WithDisableConnectionPool() RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.DisableConnectionPool = true
	}
}

// WithMultiplexed 开启连接多路复用
func WithMultiplexed(enable bool) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.EnableMultiplexed = enable
	}
}

// WithMultiplexedPool 设置自定义的连接复用池子
// 设置后开启连接复用
func WithMultiplexedPool(m *multiplexed.Multiplexed) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.EnableMultiplexed = true
		opts.Multiplexed = m
	}
}

// WithMsg 设置 msg
func WithMsg(msg codec.Msg) RoundTripOption {
	return func(opts *RoundTripOptions) {
		opts.Msg = msg
	}
}

// WithLocalAddr 建立连接时指定本地地址，多网卡时默认随机选择
func WithLocalAddr(addr string) RoundTripOption {
	return func(o *RoundTripOptions) {
		o.LocalAddr = addr
	}
}

// WithDialTimeout 设置建立连接超时
func WithDialTimeout(dur time.Duration) RoundTripOption {
	return func(o *RoundTripOptions) {
		o.DialTimeout = dur
	}
}
