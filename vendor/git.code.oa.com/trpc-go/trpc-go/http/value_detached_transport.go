package http

import (
	"net/http"
	"net/http/httptrace"
)

// newValueDetachedTransport 创建一个新的 valueDetachedTransport
func newValueDetachedTransport(r http.RoundTripper) http.RoundTripper {
	return &valueDetachedTransport{RoundTripper: r}
}

// valueDetachedTransport
// 将 ValueDetached 使用单独的 RoundTripper 实现
// 使用单独的 RoundTripper 处理 ctx，在ctx done后卸掉ctx的value
// 同时支持业务自定义 RoundTripper 在RoundTripper 中获取ctx中的value
type valueDetachedTransport struct {
	http.RoundTripper
}

// RoundTrip 实现 http.RoundTripper
func (vdt *valueDetachedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	trace := httptrace.ContextClientTrace(ctx)
	ctx = detachCtxValue(ctx)
	if trace != nil {
		ctx = httptrace.WithClientTrace(ctx, trace)
	}
	req = req.WithContext(ctx)
	return vdt.RoundTripper.RoundTrip(req)
}

// CancelRequest 实现 canceler
func (vdt *valueDetachedTransport) CancelRequest(req *http.Request) {
	// canceler 判断RoundTripper是否实现了 http.RoundTripper.CancelRequest 函数，
	// CancelRequest 是在 Go 1.5 or Go 1.6 之后支持
	type canceler interface{ CancelRequest(*http.Request) }
	if v, ok := vdt.RoundTripper.(canceler); ok {
		v.CancelRequest(req)
	}
}
