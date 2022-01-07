package loadbalance

import (
	"context"
	"time"
)

// Options 调用参数
type Options struct {
	Ctx             context.Context // 请求的 context
	Interval        time.Duration   // 列表刷新
	Namespace       string          // 命名空间
	Key             string          // hash key
	LoadBalanceType string          // 负载均衡类型
	Replicas        int             //一致性哈希算法的虚拟节点系数
}

// Option 调用参数工具函数
type Option func(*Options)

// WithContext 设置请求的 ctx
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Ctx = ctx
	}
}

// WithNamespace 设置 namespace
func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Namespace = namespace
	}
}

// WithInterval 设置负载均衡刷新列表间隔
func WithInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.Interval = interval
	}
}

// WithKey 指定有状态路由hash key
func WithKey(k string) Option {
	return func(o *Options) {
		o.Key = k
	}
}

// WithReplicas 指定一致性哈希的虚拟节点系数
func WithReplicas(r int) Option {
	return func(o *Options) {
		o.Replicas = r
	}
}

// WithLoadBalanceType 设置负载均衡类型
func WithLoadBalanceType(name string) Option {
	return func(opts *Options) {
		opts.LoadBalanceType = name
	}
}
