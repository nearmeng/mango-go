package debuglog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/log"

	trpc "git.code.oa.com/trpc-go/trpc-go"
)

func init() {
	filter.Register("debuglog", ServerFilter(), ClientFilter())
	filter.Register("simpledebuglog", ServerFilter(WithLogFunc(SimpleLogFunc)),
		ClientFilter(WithLogFunc(SimpleLogFunc)))
	filter.Register("pjsondebuglog", ServerFilter(WithLogFunc(PrettyJSONLogFunc)),
		ClientFilter(WithLogFunc(PrettyJSONLogFunc)))
	filter.Register("jsondebuglog", ServerFilter(WithLogFunc(JSONLogFunc)),
		ClientFilter(WithLogFunc(JSONLogFunc)))
}

// options 配置选项
type options struct {
	logfunc LogFunc
	exclude []*ExcludeItem
}

// Option 设置选项
type Option func(*options)

// LogFunc 结构体打印方法函数
type LogFunc func(ctx context.Context, req, rsp interface{}) string

// WithLogFunc 设置打印body方法
func WithLogFunc(f LogFunc) Option {
	return func(opts *options) {
		opts.logfunc = f
	}
}

// WithExclude 设置排除选项
func WithExclude(ex *ExcludeItem) Option {
	return func(opts *options) {
		opts.exclude = append(opts.exclude, ex)
	}
}

// DefaultLogFunc 默认结构体打印方法
var DefaultLogFunc = func(ctx context.Context, req, rsp interface{}) string {
	return fmt.Sprintf(", req:%+v, rsp:%+v", req, rsp)
}

// SimpleLogFunc 不打印结构体
var SimpleLogFunc = func(ctx context.Context, req, rsp interface{}) string {
	return ""
}

// PrettyJSONLogFunc 格式化json打印方法
var PrettyJSONLogFunc = func(ctx context.Context, req, rsp interface{}) string {
	reqJSON, _ := json.MarshalIndent(req, "", "  ")
	rspJSON, _ := json.MarshalIndent(rsp, "", "  ")
	return fmt.Sprintf("\nreq:%s\nrsp:%s", string(reqJSON), string(rspJSON))
}

// JSONLogFunc json打印方法
var JSONLogFunc = func(ctx context.Context, req, rsp interface{}) string {
	reqJSON, _ := json.Marshal(req)
	rspJSON, _ := json.Marshal(rsp)
	return fmt.Sprintf("\nreq:%s\nrsp:%s", string(reqJSON), string(rspJSON))
}

// ServerFilter 服务端过滤器
func ServerFilter(opts ...Option) filter.Filter {

	o := getFilterOptions(opts...)

	return func(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) (err error) {

		msg := trpc.Message(ctx)

		begin := time.Now()

		err = handler(ctx, req, rsp)

		for _, ex := range o.exclude {
			if ex.Method != "" && ex.Method == msg.ServerRPCName() {
				return err
			}
			if ex.Retcode != 0 && errs.Code(err) == ex.Retcode {
				return err
			}
		}

		end := time.Now()

		var addr string
		if msg.RemoteAddr() != nil {
			addr = msg.RemoteAddr().String()
		}

		if err == nil {
			log.DebugContextf(ctx, "server request:%s, cost:%s, from:%s%s",
				msg.ServerRPCName(), end.Sub(begin), addr, o.logfunc(ctx, req, rsp))
		} else {
			deadline, ok := ctx.Deadline()
			if ok {
				log.ErrorContextf(ctx, "server request:%s, cost:%s, from:%s, err:%s, total timeout:%s%s",
					msg.ServerRPCName(), end.Sub(begin), addr, err.Error(), deadline.Sub(begin), o.logfunc(ctx, req, rsp))
			} else {
				log.ErrorContextf(ctx, "server request:%s, cost:%s, from:%s, err:%s%s",
					msg.ServerRPCName(), end.Sub(begin), addr, err.Error(), o.logfunc(ctx, req, rsp))
			}
		}

		return err
	}
}

// ClientFilter 客户端过滤器
func ClientFilter(opts ...Option) filter.Filter {

	o := getFilterOptions(opts...)

	return func(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) (err error) {

		msg := trpc.Message(ctx)

		begin := time.Now()

		err = handler(ctx, req, rsp)

		for _, ex := range o.exclude {
			if ex.Retcode != 0 && errs.Code(err) == ex.Retcode {
				return err
			}
			if ex.Method != "" && ex.Method == msg.ClientRPCName() {
				return err
			}
		}

		end := time.Now()

		var addr string
		if msg.RemoteAddr() != nil {
			addr = msg.RemoteAddr().String()
		}

		if err == nil {
			log.DebugContextf(ctx, "client request:%s, cost:%s, to:%s%s",
				msg.ClientRPCName(), end.Sub(begin), addr, o.logfunc(ctx, req, rsp))
		} else {
			log.ErrorContextf(ctx, "client request:%s, cost:%s, to:%s, err:%s%s",
				msg.ClientRPCName(), end.Sub(begin), addr, err.Error(), o.logfunc(ctx, req, rsp))
		}

		return err
	}
}

// getFilterOptions 获取拦截器条件选项
func getFilterOptions(opts ...Option) *options {
	o := &options{
		logfunc: DefaultLogFunc,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
