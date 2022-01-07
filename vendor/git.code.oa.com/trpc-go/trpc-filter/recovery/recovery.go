// Package recovery tRPC过滤器,用于使服务端从panic状态恢复回来
package recovery

import (
	"context"
	"fmt"
	"runtime"

	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/metrics"
)

func init() {
	filter.Register("recovery", ServerFilter(), nil)
}

// PanicBufLen panic调用栈日志buffer大小，默认1024
var PanicBufLen = 1024

type options struct {
	rh RecoveryHandler
}

// Option 设置Recovery选项
type Option func(*options)

// RecoveryHandler recovery处理函数
type RecoveryHandler func(ctx context.Context, err interface{}) error

// WithRecoveryHandler 设置Recovery处理函数
func WithRecoveryHandler(rh RecoveryHandler) Option {
	return func(opts *options) {
		opts.rh = rh
	}
}

var defaultRecoveryHandler = func(ctx context.Context, e interface{}) error {
	buf := make([]byte, PanicBufLen)
	buf = buf[:runtime.Stack(buf, false)]
	log.ErrorContextf(ctx, "[PANIC]%v\n%s\n", e, buf)
	metrics.IncrCounter("trpc.PanicNum", 1)
	return errs.NewFrameError(errs.RetServerSystemErr, fmt.Sprint(e))
}

var defaultOptions = &options{
	rh: defaultRecoveryHandler,
}

// ServerFilter 设置服务端增加recovery
func ServerFilter(opts ...Option) filter.Filter {
	o := defaultOptions
	for _, opt := range opts {
		opt(o)
	}
	return func(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = o.rh(ctx, r)
			}
		}()

		return handler(ctx, req, rsp)
	}
}
