package http

import (
	"context"
	"errors"

	"git.code.oa.com/trpc-go/trpc-go/server"

	stdhttp "net/http"
)

// ServiceDesc descriptor for server.RegisterService
var ServiceDesc = server.ServiceDesc{
	HandlerType: nil,
}

// Handle 自定义路由注册http handler
func Handle(pattern string, h stdhttp.Handler) {
	handler := func(w stdhttp.ResponseWriter, r *stdhttp.Request) error {
		h.ServeHTTP(w, r)
		return nil
	}

	ServiceDesc.Methods = append(ServiceDesc.Methods, generateMethod(pattern, handler))
}

// HandleFunc 自定义路由注册http处理函数
func HandleFunc(pattern string, handler func(w stdhttp.ResponseWriter, r *stdhttp.Request) error) {
	ServiceDesc.Methods = append(ServiceDesc.Methods, generateMethod(pattern, handler))
}

// RegisterDefaultService register service
// 全局只使用无协议文件http服务,不能和有协议文件http服务同时使用
// 使用方式详见http/README.md
func RegisterDefaultService(s server.Service) {
	DefaultServerCodec.AutoReadBody = false
	_ = s.Register(&ServiceDesc, nil)
}

// RegisterNoProtocolService register no protocol service
// 单独注册无协议文件http服务,可同时使用有协议文件http服务
// 使用方式详见http/README.md
func RegisterNoProtocolService(s server.Service) {
	_ = s.Register(&ServiceDesc, nil)
}

// RegisterServiceMux register service with http standard mux handler, 业务自己注册路由插件
// 全局只使用无协议文件http服务,不能和有协议文件http服务同时使用
func RegisterServiceMux(s server.Service, mux stdhttp.Handler) {
	DefaultServerCodec.AutoReadBody = false

	handler := func(w stdhttp.ResponseWriter, r *stdhttp.Request) error {
		mux.ServeHTTP(w, r)
		return nil
	}

	method := generateMethod("*", handler)

	var serviceDesc = server.ServiceDesc{
		HandlerType: nil,
		Methods:     []server.Method{method},
	}

	_ = s.Register(&serviceDesc, nil)
}

// RegisterNoProtocolServiceMux register service with http standard mux handler, 业务自己注册路由插件
// 单独注册无协议文件http服务,可同时使用有协议文件http服务
func RegisterNoProtocolServiceMux(s server.Service, mux stdhttp.Handler) {
	handler := func(w stdhttp.ResponseWriter, r *stdhttp.Request) error {
		mux.ServeHTTP(w, r)
		return nil
	}

	method := generateMethod("*", handler)

	var serviceDesc = server.ServiceDesc{
		HandlerType: nil,
		Methods:     []server.Method{method},
	}

	_ = s.Register(&serviceDesc, nil)
}

// generateMethod generate server method
func generateMethod(pattern string, handler func(w stdhttp.ResponseWriter, r *stdhttp.Request) error) server.Method {
	handlerFunc := func(svr interface{}, ctx context.Context, f server.FilterFunc) (rspbody interface{}, err error) {
		filters, err := f(nil)
		if err != nil {
			return nil, err
		}

		handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
			head := Head(ctx)
			if head == nil {
				return errors.New("http Handle missing http header in context")
			}
			req := head.Request.WithContext(ctx)
			rsp := head.Response
			return handler(rsp, req)
		}

		if err := filters.Handle(ctx, nil, nil, handleFunc); err != nil {
			return nil, err
		}

		return nil, nil
	}

	return server.Method{
		Name: pattern,
		Func: handlerFunc,
	}
}
