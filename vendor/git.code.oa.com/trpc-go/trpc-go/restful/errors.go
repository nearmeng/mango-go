package restful

import (
	"context"
	"net/http"

	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/restful/errors"
	"github.com/valyala/fasthttp"
)

const (
	// MarshalErrorContent marshal error 失败时的 http response body 内容
	MarshalErrorContent = `{"code": 11, "message": "failed to marshal error"}`
)

// ErrorHandler RESTful API 错误处理
type ErrorHandler func(context.Context, http.ResponseWriter, *http.Request, error)

// FastHTTPErrorHandler fasthttp 错误处理
type FastHTTPErrorHandler func(context.Context, *fasthttp.RequestCtx, error)

// WithStatusCode 包含指定 http 返回码的错误
type WithStatusCode struct {
	StatusCode int
	Err        error
}

// Error 实现 error
func (w *WithStatusCode) Error() string {
	return w.Err.Error()
}

// httpStatusMap tRPC 错误码到 http 错误码映射
var httpStatusMap = map[int32]int{
	errs.RetServerDecodeFail:   http.StatusBadRequest,
	errs.RetServerEncodeFail:   http.StatusInternalServerError,
	errs.RetServerNoService:    http.StatusNotFound,
	errs.RetServerNoFunc:       http.StatusNotFound,
	errs.RetServerTimeout:      http.StatusGatewayTimeout,
	errs.RetServerOverload:     http.StatusTooManyRequests,
	errs.RetServerSystemErr:    http.StatusInternalServerError,
	errs.RetServerAuthFail:     http.StatusUnauthorized,
	errs.RetServerValidateFail: http.StatusBadRequest,
	errs.RetUnknown:            http.StatusInternalServerError,
}

// marshalError marshal 错误
func marshalError(err error, s Serializer) ([]byte, error) {
	// 由于 RESTful API 的 Serializer marshalling 都是针对 proto message
	// 所以先把 tRPC error 转成 proto message 格式表示
	terr := &errors.Err{
		Code:    int32(errs.Code(err)),
		Message: errs.Msg(err),
	}

	return s.Marshal(terr)
}

// statusCodeFromError 根据错误获取响应头
func statusCodeFromError(err error) int {
	statusCode := http.StatusInternalServerError

	if withStatusCode, ok := err.(*WithStatusCode); ok {
		statusCode = withStatusCode.StatusCode
	} else {
		if statusFromMap, ok := httpStatusMap[int32(errs.Code(err))]; ok {
			statusCode = statusFromMap
		}
	}

	return statusCode
}

// defaultErrorHandler 默认错误处理
var defaultErrorHandler = func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	// 获取 Serializer
	_, s := serializerForTranscoding(r.Header[headerContentType],
		r.Header[headerAccept])

	// marshal 错误
	buf, merr := marshalError(err, s)
	if merr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(MarshalErrorContent))
		return
	}
	// 回包
	w.WriteHeader(statusCodeFromError(err))
	w.Write(buf)
}

// fastHTTPErrorHandler 默认的基于 fasthttp 实现时的错误处理
var fastHTTPErrorHandler = func(ctx context.Context, requestCtx *fasthttp.RequestCtx, err error) {
	// 获取 Serializer
	_, s := serializerForTranscoding(
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerContentType))},
		[]string{bytes2str(requestCtx.Request.Header.Peek(headerAccept))},
	)

	// marshal 错误
	buf, merr := marshalError(err, s)
	if merr != nil {
		requestCtx.Response.SetStatusCode(http.StatusInternalServerError)
		requestCtx.Write([]byte(MarshalErrorContent))
		return
	}
	// 回包
	requestCtx.SetStatusCode(statusCodeFromError(err))
	requestCtx.Write(buf)
}
