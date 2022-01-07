package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/naming/circuitbreaker"
	"git.code.oa.com/trpc-go/trpc-go/naming/discovery"
	"git.code.oa.com/trpc-go/trpc-go/naming/loadbalance"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/naming/selector"
	"git.code.oa.com/trpc-go/trpc-go/overloadctrl"
	"git.code.oa.com/trpc-go/trpc-go/pool/connpool"
	"git.code.oa.com/trpc-go/trpc-go/pool/multiplexed"
	"git.code.oa.com/trpc-go/trpc-go/transport"
)

// Options 客户端调用参数
type Options struct {
	ServiceName       string        // 后端服务service name
	CallerServiceName string        // 调用服务service name 即server自身服务名
	CalleeMethod      string        // 用于监控上报的method方法名
	Timeout           time.Duration // 后端调用超时时间

	Target   string // 后端服务地址 name://endpoint 兼容老寻址方式 如 cl5://sid cmlb://appid ip://ip:port
	endpoint string // 默认等于 service name，除非有指定target

	OverloadCtrl overloadctrl.OverloadController // 客户端过载保护

	Network           string
	CallType          codec.RequestType           // 请求类型，取值参考 transport.RequestType
	CallOptions       []transport.RoundTripOption // client transport需要调用的参数
	Transport         transport.ClientTransport
	EnableMultiplexed bool
	StreamTransport   transport.ClientStreamTransport

	SelectOptions        []selector.Option
	Selector             selector.Selector
	DisableServiceRouter bool

	CurrentSerializationType int
	CurrentCompressType      int
	SerializationType        int
	CompressType             int

	Codec                 codec.Codec
	MetaData              codec.MetaData
	ClientStreamQueueSize int // 客户端流式接收的缓冲区大小

	Filters       filter.Chain // 链式拦截器
	DisableFilter bool         // 是否禁用拦截器

	ReqHead interface{} // 提供用户设置自定义请求头的能力
	RspHead interface{} // 提供用户获取自定义响应头的能力
	Node    *onceNode   // 提供用户获取具体请求节点的能力

	MaxWindowSize uint32      // 设置客户端流式接收端最大的window大小
	SControl      SendControl // 流控发送端控制
	RControl      RecvControl // 流控接收端统计
}

type onceNode struct {
	*registry.Node
	once sync.Once
}

func (n *onceNode) set(serviceName, address string, cost time.Duration) {
	if n == nil {
		return
	}
	n.once.Do(func() {
		n.Node.ServiceName = serviceName
		n.Node.Address = address
		n.Node.CostTime = cost
	})
}

// Option 调用参数工具函数
type Option func(*Options)

// WithNamespace 设置 namespace 后端服务环境 正式环境 Production 测试环境 Development
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithNamespace(namespace))
	}
}

// WithClientStreamQueueSize 客户端流式缓冲区大小，接收到消息后channel可以放置的消息数
func WithClientStreamQueueSize(size int) Option {
	return func(o *Options) {
		o.ClientStreamQueueSize = size
	}
}

// WithServiceName 设置后端服务service name
func WithServiceName(s string) Option {
	return func(o *Options) {
		o.ServiceName = s
		o.endpoint = s
	}
}

// WithCallerServiceName 设置主调服务service name, 即自身服务的service name
func WithCallerServiceName(s string) Option {
	return func(o *Options) {
		o.CallerServiceName = s
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceServiceName(s))
	}
}

// WithCallerNamespace 设置主调服namespace, 即自身服务 namespace
func WithCallerNamespace(s string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceNamespace(s))
	}
}

// WithDisableFilter 禁用拦截器, 假如插件里面setup时需要使用client，但这时候filter都还没初始化，此时就可以先禁用拦截器
func WithDisableFilter() Option {
	return func(o *Options) {
		o.DisableFilter = true
	}
}

// WithDisableServiceRouter 禁用服务路由
func WithDisableServiceRouter() Option {
	return func(o *Options) {
		o.DisableServiceRouter = true
		o.SelectOptions = append(o.SelectOptions, selector.WithDisableServiceRouter())
	}
}

// WithEnvKey 设置环境key
func WithEnvKey(key string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithEnvKey(key))
	}
}

// WithCallerEnvName 设置当前环境
func WithCallerEnvName(envName string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceEnvName(envName))
	}
}

// WithCallerSetName 设置调用者set分组
func WithCallerSetName(setName string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceSetName(setName))
	}
}

// WithCalleeSetName 指定set分组调用
func WithCalleeSetName(setName string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithDestinationSetName(setName))
	}
}

// WithCalleeEnvName 设置被调服务环境
func WithCalleeEnvName(envName string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithDestinationEnvName(envName))
	}
}

// WithCalleeMethod 指定下游方法名
func WithCalleeMethod(method string) Option {
	return func(o *Options) {
		o.CalleeMethod = method
	}
}

// WithCallerMetadata 增加主调服务的路由匹配元数据，env/set路由请使用相应配置函数
func WithCallerMetadata(key string, val string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithSourceMetadata(key, val))
	}
}

// WithCalleeMetadata 增加被调服务的路由匹配元数据，env/set路由请使用相应配置函数
func WithCalleeMetadata(key string, val string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithDestinationMetadata(key, val))
	}
}

// WithBalancerName 通过名字指定负载均衡
func WithBalancerName(balancerName string) Option {
	balancer := loadbalance.Get(balancerName)
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions,
			selector.WithLoadBalancer(balancer),
			selector.WithLoadBalanceType(balancerName),
		)
	}
}

// WithDiscoveryName 通过名字指定名字服务
func WithDiscoveryName(name string) Option {
	d := discovery.Get(name)
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithDiscovery(d))
	}
}

// WithCircuitBreakerName 通过名字指定熔断器
func WithCircuitBreakerName(name string) Option {
	cb := circuitbreaker.Get(name)
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithCircuitBreaker(cb))
	}
}

// WithKey 设置有状态的路由key
func WithKey(key string) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithKey(key))
	}
}

// WithReplicas 设置有状态的路由的节点副本数
func WithReplicas(r int) Option {
	return func(o *Options) {
		o.SelectOptions = append(o.SelectOptions, selector.WithReplicas(r))
	}
}

// WithTarget 调用目标地址schema name://endpoint 如 cl5://sid ons://zkname ip://ip:port
func WithTarget(t string) Option {
	return func(o *Options) {
		o.Target = t
	}
}

// WithNetwork 对端服务网络类型 tcp or udp, 默认tcp
func WithNetwork(s string) Option {
	return func(o *Options) {
		o.Network = s
		o.CallOptions = append(o.CallOptions, transport.WithDialNetwork(s))
	}
}

// WithPassword 对端服务请求密码
func WithPassword(s string) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithDialPassword(s))
	}
}

// WithPool 请求后端时 自定义tcp连接池
func WithPool(pool connpool.Pool) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithDialPool(pool))
	}
}

// WithMultiplexedPool 设置自定义连接复用池
// 一旦设置自定义连接复用池，代表启用连接复用
func WithMultiplexedPool(m *multiplexed.Multiplexed) Option {
	return func(o *Options) {
		o.EnableMultiplexed = true
		o.CallOptions = append(o.CallOptions, transport.WithMultiplexedPool(m))
	}
}

// WithTimeout 请求后端超时时间
func WithTimeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// WithCurrentSerializationType 设置当前请求序列化方式，指定后端协议内部序列化方式，使用 WithSerializationType
func WithCurrentSerializationType(t int) Option {
	return func(o *Options) {
		o.CurrentSerializationType = t
	}
}

// WithSerializationType 指定后端协议内部序列化方式，一般只需指定该option，current用于代理转发层
func WithSerializationType(t int) Option {
	return func(o *Options) {
		o.SerializationType = t
	}
}

// WithCurrentCompressType 设置当前请求解压缩方式，指定后端协议内部解压缩方式，使用 WithCompressType
func WithCurrentCompressType(t int) Option {
	return func(o *Options) {
		o.CurrentCompressType = t
	}
}

// WithCompressType 指定后端协议内部解压缩方式，一般只需指定该option，current用于代理转发层
func WithCompressType(t int) Option {
	return func(o *Options) {
		o.CompressType = t
	}
}

// WithTransport 替换底层client通信层
func WithTransport(t transport.ClientTransport) Option {
	return func(o *Options) {
		o.Transport = t
	}
}

// WithProtocol 指定后端服务协议名字 如 trpc
func WithProtocol(s string) Option {
	return func(o *Options) {
		o.Codec = codec.GetClient(s)
		r := transport.GetFramerBuilder(s)
		if r != nil {
			o.CallOptions = append(o.CallOptions, transport.WithClientFramerBuilder(r))
		}
		trans := transport.GetClientTransport(s)
		if trans != nil {
			o.Transport = trans
		}
	}
}

// WithConnectionMode 设置连接是否为connected模式（connected限制udp只收相同路径回包）
func WithConnectionMode(connMode transport.ConnectionMode) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithConnectionMode(connMode))
	}
}

// WithSendOnly 设置只发不收，一般用于udp异步发送
func WithSendOnly() Option {
	return func(o *Options) {
		o.CallType = codec.SendOnly
		o.CallOptions = append(o.CallOptions, transport.WithReqType(codec.SendOnly))
	}
}

// WithFilter 添加客户端拦截器，支持在 打包前 解包后 拦截处理
func WithFilter(fs filter.Filter) Option {
	return func(o *Options) {
		o.Filters = append(o.Filters, fs)
	}
}

// WithFilters 添加客户端拦截器，支持在 打包前 打包后 拦截处理
func WithFilters(fs []filter.Filter) Option {
	return func(o *Options) {
		o.Filters = append(o.Filters, fs...)
	}
}

// WithReqHead 设置后端请求包头，可不设置，默认会从请求源头clone server req head
func WithReqHead(h interface{}) Option {
	return func(o *Options) {
		o.ReqHead = h
	}
}

// WithRspHead 设置后端响应包头，不关心时可不设置, 一般用于网关服务
func WithRspHead(h interface{}) Option {
	return func(o *Options) {
		o.RspHead = h
	}
}

// WithMetaData 设置透传参数
func WithMetaData(key string, val []byte) Option {
	return func(o *Options) {
		if o.MetaData == nil {
			o.MetaData = codec.MetaData{}
		}
		o.MetaData[key] = val
	}
}

// WithSelectorNode 设置后端selector寻址node结果保存器，不关心时可不设置, 常用于定位问题节点
func WithSelectorNode(n *registry.Node) Option {
	return func(o *Options) {
		o.Node = &onceNode{Node: n}
	}
}

// SetNamingOptions 设置寻址相关 option
func (opts *Options) SetNamingOptions(cfg *BackendConfig) error {
	if cfg.ServiceName != "" {
		opts.ServiceName = cfg.ServiceName
		opts.endpoint = cfg.ServiceName
	}
	if cfg.Namespace != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithNamespace(cfg.Namespace))
	}
	if cfg.EnvName != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDestinationEnvName(cfg.EnvName))
	}
	if cfg.SetName != "" {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDestinationSetName(cfg.SetName))
	}
	if cfg.DisableServiceRouter {
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDisableServiceRouter())
		opts.DisableServiceRouter = true
	}

	if cfg.Target != "" {
		opts.Target = cfg.Target
		return nil
	}
	if cfg.Discovery != "" {
		d := discovery.Get(cfg.Discovery)
		if d == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: discovery %s no registered", cfg.Discovery))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithDiscovery(d))
	}
	if cfg.Loadbalance != "" {
		balancer := loadbalance.Get(cfg.Loadbalance)
		if balancer == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: balancer %s no registered", cfg.Loadbalance))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithLoadBalancer(balancer))
	}
	if cfg.Circuitbreaker != "" {
		cb := circuitbreaker.Get(cfg.Circuitbreaker)
		if cb == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: circuitbreaker %s no registered", cfg.Circuitbreaker))
		}
		opts.SelectOptions = append(opts.SelectOptions, selector.WithCircuitBreaker(cb))
	}
	return nil
}

// LoadClientConfig 通过key读取后端配置, key默认为proto协议文件里面的callee service name
func (opts *Options) LoadClientConfig(key string) error {
	cfg := Config(key)
	if err := opts.SetNamingOptions(cfg); err != nil {
		return err
	}

	opts.OverloadCtrl = &cfg.OverloadCtrl
	if cfg.Timeout > 0 {
		opts.Timeout = time.Duration(cfg.Timeout) * time.Millisecond
	}
	if cfg.Serialization != nil {
		opts.SerializationType = *cfg.Serialization
	}
	if cfg.Compression > 0 {
		opts.CompressType = cfg.Compression
	}
	if cfg.Protocol != "" {
		o := WithProtocol(cfg.Protocol)
		o(opts)
	}
	if cfg.Network != "" {
		opts.Network = cfg.Network
		opts.CallOptions = append(opts.CallOptions, transport.WithDialNetwork(cfg.Network))
	}
	if cfg.Password != "" {
		opts.CallOptions = append(opts.CallOptions, transport.WithDialPassword(cfg.Password))
	}
	if cfg.CACert != "" {
		opts.CallOptions = append(opts.CallOptions,
			transport.WithDialTLS(cfg.TLSCert, cfg.TLSKey, cfg.CACert, cfg.TLSServerName))
	}
	return nil
}

// LoadClientFilterConfig 通过key读取后端Filter配置
func (opts *Options) LoadClientFilterConfig(key string) error {
	if opts.DisableFilter {
		opts.Filters = filter.EmptyChain
		return nil
	}
	cfg := Config(key)
	for _, filterName := range cfg.Filter {
		f := filter.GetClient(filterName)
		if f == nil {
			return errs.NewFrameError(errs.RetServerSystemErr,
				fmt.Sprintf("client config: filter %s no registered", filterName))
		}
		opts.Filters = append(opts.Filters, f)
	}
	return nil
}

// LoadNodeConfig 通过注册中心返回的节点信息设置参数
func (opts *Options) LoadNodeConfig(node *registry.Node) {
	opts.CallOptions = append(opts.CallOptions, transport.WithDialAddress(node.Address))
	// 名字服务优先。当名字服务没有network时，以本地配置为准
	if node.Network != "" {
		opts.Network = node.Network
		opts.CallOptions = append(opts.CallOptions, transport.WithDialNetwork(node.Network))
	} else {
		node.Network = opts.Network
	}

	if node.Protocol != "" {
		o := WithProtocol(node.Protocol)
		o(opts)
	}
}

// WithTLS 指定client tls文件地址, caFile CA证书，用于校验server证书, 一般调用https只需要指定caFile即可。
// 也可以传入caFile="none"表示不校验server证书, caFile="root"表示使用本机安装的ca证书来验证server。
// certFile客户端自身证书，keyFile客户端自身秘钥，服务端开启双向认证需要校验客户端证书时才需要发送客户端自身证书，一般为空即可。
// serverName客户端校验服务端的服务名，https可为空默认为hostname。
func WithTLS(certFile, keyFile, caFile, serverName string) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithDialTLS(certFile, keyFile, caFile, serverName))
	}
}

// WithDisableConnectionPool 禁用连接池
func WithDisableConnectionPool() Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithDisableConnectionPool())
	}
}

// WithMultiplexed 开启连接多路复用
// 如果要自定义Multiplexed的参数，请使用 WithMultiplexedPool 进行设置
func WithMultiplexed(enable bool) Option {
	return func(o *Options) {
		o.EnableMultiplexed = enable
	}
}

// WithLocalAddr 建立连接时指定本地地址，多网卡时默认随机选择。
//
// 短连接模式下可以指定地址和端口：
// client.WithLocalAddr("127.0.0.1:8080")
// 连接池和连接复用模式下同一个配置会建立多个连接只能指定地址:
// client.WithLocalAddr("127.0.0.1:")
func WithLocalAddr(addr string) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithLocalAddr(addr))
	}
}

// WithDialTimeout 设置建立连接超时
func WithDialTimeout(dur time.Duration) Option {
	return func(o *Options) {
		o.CallOptions = append(o.CallOptions, transport.WithDialTimeout(dur))
	}
}

// WithStreamTransport 传入流式的 transport
func WithStreamTransport(st transport.ClientStreamTransport) Option {
	return func(o *Options) {
		o.StreamTransport = st
	}
}

// WithOverloadCtrl 设置客户端过载保护策略。
func WithOverloadCtrl(oc overloadctrl.OverloadController) Option {
	return func(o *Options) {
		o.OverloadCtrl = oc
	}
}

// WithMaxWindowSize 设置最大接收窗口的大小
// client作为接收方会将此窗口公告给接收方
func WithMaxWindowSize(s uint32) Option {
	return func(o *Options) {
		o.MaxWindowSize = s
	}
}

// WithSendControl 设置流控接收窗口的大小
func WithSendControl(sc SendControl) Option {
	return func(o *Options) {
		o.SControl = sc
	}
}

// WithRecvControl 设置流控接收窗口的大小
func WithRecvControl(rc RecvControl) Option {
	return func(o *Options) {
		o.RControl = rc
	}
}

type optionsImmutability struct{}

// WithOptionsImmutable 标记最外层的 Options 是 immutable 的。
// 如果下层需要修改 Options，必须先拷贝一个新的出来。
//
// 该方法是提供给拦截器使用的（一些拦截器可能会并发地调用下一个拦截器），
// 普通用户不应该在应用层调用它。
func WithOptionsImmutable(ctx context.Context) context.Context {
	return context.WithValue(ctx, optionsImmutability{}, optionsImmutability{})
}

// IsOptionsImmutable 检查 ctx 是否将 Options 指定为 immutable 的。
func IsOptionsImmutable(ctx context.Context) bool {
	_, ok := ctx.Value(optionsImmutability{}).(optionsImmutability)
	return ok
}

func mutateOptions(ctx context.Context, options *Options) *Options {
	if IsOptionsImmutable(ctx) {
		// 原始 options 是不可变的，返回一个它的拷贝。
		opts := *options
		return &opts
	}
	return options
}
