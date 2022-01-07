package discovery

import (
	"context"
)

// Options 调用参数
type Options struct {
	Ctx       context.Context
	Namespace string
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
