package selector

import (
	"context"

	"git.code.oa.com/trpc-go/trpc-go/naming/circuitbreaker"
	"git.code.oa.com/trpc-go/trpc-go/naming/discovery"
	"git.code.oa.com/trpc-go/trpc-go/naming/loadbalance"
	"git.code.oa.com/trpc-go/trpc-go/naming/servicerouter"
)

var (
	defaultDiscoveryOptionsSize     = 2
	defaultServiceRouterOptionsSize = 2
	defaultLoadBalanceOptionsSize   = 2
)

// Options 调用参数
type Options struct {
	// Ctx 与请求相关联的 Context
	Ctx context.Context
	// 有状态路由的 hash key
	Key string
	// 有状态路由的单节点副本数(可选，用于hash环寻址)
	Replicas int
	// EnvKey 设置的环境key
	EnvKey string
	// Namespace 被调服务命名空间
	Namespace string
	// SourceNamespace 主调服务命名空间
	SourceNamespace string
	// SourceServiceName 主调服务名
	SourceServiceName string
	// SourceEnvName  主调服务环境名
	SourceEnvName string
	// SourceSetname 主调服务的set分组
	SourceSetName string
	// SourceMetadata 主调服务的路由匹配元数据
	SourceMetadata map[string]string
	// DestinationEnvName 被调服务环境名，用于获取指定环境节点
	DestinationEnvName string
	// DestinationSetName 指定set调用
	DestinationSetName string
	// DestinationMetadata 被调服务的路由匹配元数据
	DestinationMetadata map[string]string
	// LoadBalanceType 负载均衡类型
	LoadBalanceType string

	// EnvTransfer 上游服务透传的环境信息
	EnvTransfer          string
	Discovery            discovery.Discovery
	DiscoveryOptions     []discovery.Option
	ServiceRouter        servicerouter.ServiceRouter
	ServiceRouterOptions []servicerouter.Option
	LoadBalancer         loadbalance.LoadBalancer
	LoadBalanceOptions   []loadbalance.Option
	CircuitBreaker       circuitbreaker.CircuitBreaker
	DisableServiceRouter bool
}

// Option 调用参数工具函数
type Option func(*Options)

// WithContext 设置请求的 ctx
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Ctx = ctx
		o.DiscoveryOptions = append(o.DiscoveryOptions, discovery.WithContext(ctx))
		o.LoadBalanceOptions = append(o.LoadBalanceOptions, loadbalance.WithContext(ctx))
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithContext(ctx))
	}
}

// WithNamespace 设置 namespace
func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Namespace = namespace
		opts.DiscoveryOptions = append(opts.DiscoveryOptions, discovery.WithNamespace(namespace))
		opts.LoadBalanceOptions = append(opts.LoadBalanceOptions, loadbalance.WithNamespace(namespace))
		opts.ServiceRouterOptions = append(opts.ServiceRouterOptions, servicerouter.WithNamespace(namespace))
	}
}

// WithSourceSetName 指定路由set
func WithSourceSetName(sourceSetName string) Option {
	return func(opts *Options) {
		opts.SourceSetName = sourceSetName
		opts.ServiceRouterOptions = append(opts.ServiceRouterOptions, servicerouter.WithSourceSetName(sourceSetName))
	}
}

// WithKey 指定有状态路由hash key
func WithKey(k string) Option {
	return func(o *Options) {
		o.Key = k
		o.LoadBalanceOptions = append(o.LoadBalanceOptions, loadbalance.WithKey(k))
	}
}

// WithReplicas 设置有状态路由节点副本数
func WithReplicas(r int) Option {
	return func(o *Options) {
		o.Replicas = r
		o.LoadBalanceOptions = append(o.LoadBalanceOptions, loadbalance.WithReplicas(r))
	}
}

// WithDisableServiceRouter 禁用服务路由
func WithDisableServiceRouter() Option {
	return func(o *Options) {
		o.DisableServiceRouter = true
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithDisableServiceRouter())
	}
}

// WithDiscovery 指定服务发现
func WithDiscovery(d discovery.Discovery) Option {
	return func(o *Options) {
		o.Discovery = d
	}
}

// WithLoadBalance 指定负载均衡
// Deprecated: 使用 WithLoadBalancer 替代
func WithLoadBalance(b loadbalance.LoadBalancer) Option {
	return func(o *Options) {
		o.LoadBalancer = b
	}
}

// WithLoadBalancer 指定负载均衡器
func WithLoadBalancer(b loadbalance.LoadBalancer) Option {
	return func(o *Options) {
		o.LoadBalancer = b
	}
}

// WithLoadBalanceType 指定负载均衡类型
func WithLoadBalanceType(name string) Option {
	return func(o *Options) {
		o.LoadBalanceType = name
		o.LoadBalanceOptions = append(
			o.LoadBalanceOptions,
			loadbalance.WithLoadBalanceType(name),
		)
	}
}

// WithCircuitBreaker 指定熔断器
func WithCircuitBreaker(cb circuitbreaker.CircuitBreaker) Option {
	return func(o *Options) {
		o.CircuitBreaker = cb
	}
}

// WithEnvKey 指定环境key路由
func WithEnvKey(key string) Option {
	return func(o *Options) {
		o.EnvKey = key
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithEnvKey(key))
	}
}

// WithSourceNamespace 指定源服务 namespace
func WithSourceNamespace(namespace string) Option {
	return func(o *Options) {
		o.SourceNamespace = namespace
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithSourceNamespace(namespace))
	}
}

// WithSourceServiceName 指定源服务名
func WithSourceServiceName(serviceName string) Option {
	return func(o *Options) {
		o.SourceServiceName = serviceName
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithSourceServiceName(serviceName))
	}
}

// WithDestinationEnvName 指定被调服务环境
func WithDestinationEnvName(envName string) Option {
	return func(o *Options) {
		o.DestinationEnvName = envName
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithDestinationEnvName(envName))
	}
}

// WithSourceEnvName 指定源服务环境
func WithSourceEnvName(envName string) Option {
	return func(o *Options) {
		o.SourceEnvName = envName
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithSourceEnvName(envName))
	}
}

// WithEnvTransfer 指定透传环境信息
func WithEnvTransfer(envTransfer string) Option {
	return func(o *Options) {
		o.EnvTransfer = envTransfer
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithEnvTransfer(envTransfer))
	}
}

// WithDestinationSetName 指定
func WithDestinationSetName(destinationSetName string) Option {
	return func(o *Options) {
		o.DestinationSetName = destinationSetName
		o.ServiceRouterOptions = append(o.ServiceRouterOptions,
			servicerouter.WithDestinationSetName(destinationSetName))
	}
}

// WithSourceMetadata 增加主调服务的路由匹配元数据，env/set路由请使用相应配置函数
func WithSourceMetadata(key string, val string) Option {
	return func(o *Options) {
		if o.SourceMetadata == nil {
			o.SourceMetadata = make(map[string]string)
		}
		o.SourceMetadata[key] = val
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithSourceMetadata(key, val))
	}
}

// WithDestinationMetadata 增加被调服务的路由匹配元数据，env/set路由请使用相应配置函数
func WithDestinationMetadata(key string, val string) Option {
	return func(o *Options) {
		if o.DestinationMetadata == nil {
			o.DestinationMetadata = make(map[string]string)
		}
		o.DestinationMetadata[key] = val
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithDestinationMetadata(key, val))
	}
}
