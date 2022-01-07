package restful

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/restful/dat"
	"google.golang.org/protobuf/proto"
)

// Router RESTful API 路由
type Router struct {
	opts        *Options
	transcoders map[string][]*transcoder
}

// NewRouter 创建一个路由
func NewRouter(opt ...Option) *Router {
	opts := &Options{
		ErrorHandler:          defaultErrorHandler,
		HeaderMatcher:         defaultHeaderMatcher,
		ResponseHandler:       defaultResponseHandler,
		FastHTTPErrHandler:    fastHTTPErrorHandler,
		FastHTTPHeaderMatcher: defaultFastHTTPHeaderMatcher,
		FastHTTPRespHandler:   defaultFastHTTPRespHandler,
	}

	for _, o := range opt {
		o(opts)
	}

	return &Router{
		opts:        opts,
		transcoders: make(map[string][]*transcoder),
	}
}

var (
	routers    = make(map[string]http.Handler) // tRPC service name -> Router
	routerLock sync.RWMutex
)

// RegisterRouter 注册 tRPC service 对应的 Router
func RegisterRouter(name string, router http.Handler) {
	routerLock.Lock()
	routers[name] = router
	routerLock.Unlock()
}

// GetRouter 获取 tRPC service 对应的 Router
func GetRouter(name string) http.Handler {
	routerLock.RLock()
	router := routers[name]
	routerLock.RUnlock()
	return router
}

// ProtoMessage proto.Message 别名
type ProtoMessage proto.Message

// Initializer 初始化 ProtoMessage
type Initializer func() ProtoMessage

// BodyLocator 根据 HttpRule body 定位到要 unmarshal 到 proto message 中的位置
type BodyLocator interface {
	Body() string
	Locate(ProtoMessage) interface{}
}

// ResponseBodyLocator 根据 HttpRule response_body 定位到要 marshal 的 proto message 中的位置
type ResponseBodyLocator interface {
	ResponseBody() string
	Locate(ProtoMessage) interface{}
}

// HandleFunc tRPC method 处理函数
type HandleFunc func(svc interface{}, ctx context.Context, reqbody, respbody interface{}) error

// ExtractFilterFunc 用于从 tRPC service 提取 filter chain
type ExtractFilterFunc func() filter.Chain

// Binding tRPC method 和 HttpRule 绑定
type Binding struct {
	Name         string
	Input        Initializer
	Output       Initializer
	Handler      HandleFunc
	HTTPMethod   string
	Pattern      *Pattern
	Body         BodyLocator
	ResponseBody ResponseBodyLocator
}

// AddBinding 新增一个 tRPC method 和 HttpRule 的绑定
func (r *Router) AddBinding(binding *Binding) error {
	// 创建一个 transcoder
	tr := &transcoder{
		name:                 binding.Name,
		input:                binding.Input,
		output:               binding.Output,
		handler:              binding.Handler,
		httpMethod:           binding.HTTPMethod,
		pat:                  binding.Pattern,
		body:                 binding.Body,
		respBody:             binding.ResponseBody,
		router:               r,
		discardUnknownParams: r.opts.DiscardUnknownParams,
	}

	// 新建 dat，过滤被 HttpRule 引用的字段
	var fps [][]string
	if fromPat := binding.Pattern.FieldPaths(); fromPat != nil {
		fps = append(fps, fromPat...)
	}
	if binding.Body != nil {
		if fromBody := binding.Body.Body(); fromBody != "" && fromBody != "*" {
			fps = append(fps, strings.Split(fromBody, "."))
		}
	}
	if len(fps) > 0 {
		doubleArrayTrie, err := dat.Build(fps)
		if err != nil {
			return fmt.Errorf("failed to build dat: %w", err)
		}
		tr.dat = doubleArrayTrie
	}

	// 添加 transcoder
	r.transcoders[binding.HTTPMethod] = append(r.transcoders[binding.HTTPMethod], tr)

	return nil
}

// ctxForCompatibility 仅用于兼容 thttp
var ctxForCompatibility func(context.Context, http.ResponseWriter, *http.Request) context.Context

// SetCtxForCompatibility 仅用于兼容 thttp
func SetCtxForCompatibility(f func(context.Context, http.ResponseWriter, *http.Request) context.Context) {
	ctxForCompatibility = f
}

// HeaderMatcher 从 http request 头映射到 tRPC Stub Context
type HeaderMatcher func(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	serviceName, methodName string,
) (context.Context, error)

// defaultHeaderMatcher 默认的 HeaderMatcher
var defaultHeaderMatcher = func(
	ctx context.Context,
	w http.ResponseWriter,
	req *http.Request,
	serviceName, methodName string,
) (context.Context, error) {
	// 建议：用户自定义也最好往 ctx 里面塞 codec.Msg，并且指定目标 service 和 method 名
	return withNewMessage(ctx, serviceName, methodName), nil
}

// withNewMessage 往 ctx 里面塞 codec.Msg，并且指定目标 service 和 method 名
func withNewMessage(ctx context.Context, serviceName, methodName string) context.Context {
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithServerRPCName(methodName)
	msg.WithCalleeServiceName(serviceName)
	msg.WithSerializationType(codec.SerializationTypePB)
	return ctx
}

// CustomResponseHandler 用户自定义的回包处理
type CustomResponseHandler func(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	resp proto.Message,
	body []byte,
) error

var httpStatusKey = "t-http-status"

// SetStatusCodeOnSucceed 设置处理成功时的响应码，应该为 2XX
// 如果要设置错误码不要用这个，用 restful/errors.go 的 &WithStatusCode{}
func SetStatusCodeOnSucceed(ctx context.Context, code int) {
	msg := codec.Message(ctx)
	metadata := msg.ServerMetaData()
	if metadata == nil {
		metadata = codec.MetaData{}
	}
	metadata[httpStatusKey] = []byte(strconv.Itoa(code))
	msg.WithServerMetaData(metadata)
}

// GetStatusCodeOnSucceed 获取处理成功时的响应码，必须要先在 tRPC 方法中用 SetStatusCodeOnSucceed 设置
func GetStatusCodeOnSucceed(ctx context.Context) int {
	if metadata := codec.Message(ctx).ServerMetaData(); metadata != nil {
		if buf, ok := metadata[httpStatusKey]; ok {
			if code, err := strconv.Atoi(bytes2str(buf)); err == nil {
				return code
			}
		}
	}
	return http.StatusOK
}

// defaultResponseHandler 默认的 CustomResponseHandler
var defaultResponseHandler = func(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	resp proto.Message,
	body []byte,
) error {
	// 压缩
	var writer io.Writer = w
	_, c := compressorForTranscoding(r.Header[headerContentEncoding],
		r.Header[headerAcceptEncoding])
	if c != nil {
		writeCloser, err := c.Compress(w)
		if err != nil {
			return fmt.Errorf("failed to compress resp body: %w", err)
		}
		defer writeCloser.Close()
		w.Header().Set(headerContentEncoding, c.ContentEncoding())
		writer = writeCloser
	}

	// 设置响应码
	statusCode := GetStatusCodeOnSucceed(ctx)
	w.WriteHeader(statusCode)

	// 设置 body
	if statusCode != http.StatusNoContent && statusCode != http.StatusNotModified {
		writer.Write(body)
	}

	return nil
}

// putBackCtxMessage 若 ctx 里塞了 codec.Msg，则调用 codec.PutBackMessage 放回池里
func putBackCtxMessage(ctx context.Context) {
	if msg, ok := ctx.Value(codec.ContextKeyMessage).(codec.Msg); ok {
		codec.PutBackMessage(msg)
	}
}

// ServeHTTP 实现 http.Handler
// TODO: 完善路由分发功能
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := ctxForCompatibility(req.Context(), w, req)
	for _, tr := range r.transcoders[req.Method] {
		fieldValues, err := tr.pat.Match(req.URL.Path)
		if err == nil {
			// 头转换
			stubCtx, err := r.opts.HeaderMatcher(ctx, w, req, r.opts.ServiceName, tr.name)
			if err != nil {
				r.opts.ErrorHandler(ctx, w, req, errs.New(errs.RetServerDecodeFail, err.Error()))
				return
			}

			// 获取请求 Compressor 和 Serializer
			reqCompressor, respCompressor := compressorForTranscoding(req.Header[headerContentEncoding],
				req.Header[headerAcceptEncoding])
			reqSerializer, respSerializer := serializerForTranscoding(req.Header[headerContentType],
				req.Header[headerAccept])

			// 设置转码参数
			params := paramsPool.Get().(*transcodeParams)
			params.reqCompressor = reqCompressor
			params.respCompressor = respCompressor
			params.reqSerializer = reqSerializer
			params.respSerializer = respSerializer
			params.body = req.Body
			params.fieldValues = fieldValues
			params.form = req.URL.Query()

			// 转码
			resp, body, err := tr.transcode(stubCtx, params)
			if err != nil {
				r.opts.ErrorHandler(stubCtx, w, req, err)
				putBackCtxMessage(stubCtx)
				putBackParams(params)
				return
			}

			// response content-type 设置
			w.Header().Set(headerContentType, respSerializer.ContentType())

			// 用户自定义回包
			if err := r.opts.ResponseHandler(stubCtx, w, req, resp, body); err != nil {
				r.opts.ErrorHandler(stubCtx, w, req, errs.New(errs.RetServerEncodeFail, err.Error()))
			}
			putBackCtxMessage(stubCtx)
			putBackParams(params)
			return
		}
	}
	r.opts.ErrorHandler(ctx, w, req, errs.New(errs.RetServerNoFunc, "failed to match any pattern"))
}
