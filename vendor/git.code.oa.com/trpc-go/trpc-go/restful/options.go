package restful

// Options 路由 option 选项
type Options struct {
	ServiceName           string                // tRPC service 名
	ServiceImpl           interface{}           // tRPC service 实现
	FilterFunc            ExtractFilterFunc     // 提取 tRPC service 处理链函数
	ErrorHandler          ErrorHandler          // 错误处理
	HeaderMatcher         HeaderMatcher         // 请求头映射
	ResponseHandler       CustomResponseHandler // 自定义回包处理
	FastHTTPErrHandler    FastHTTPErrorHandler  // fasthttp 错误处理
	FastHTTPHeaderMatcher FastHTTPHeaderMatcher // fasthttp 请求头映射
	FastHTTPRespHandler   FastHTTPRespHandler   // fasthttp 自定义回包处理
	DiscardUnknownParams  bool                  // 忽略未知参数的 query params
}

// Option 调用参数工具函数
type Option func(*Options)

// WithServiceName 设置 tRPC service 名
func WithServiceName(name string) Option {
	return func(opts *Options) {
		opts.ServiceName = name
	}
}

// WithServiceImpl 设置 tRPC service 具体实现
func WithServiceImpl(impl interface{}) Option {
	return func(opts *Options) {
		opts.ServiceImpl = impl
	}
}

// WithFilterFunc 设置提取 tRPC service 处理链函数
func WithFilterFunc(f ExtractFilterFunc) Option {
	return func(opts *Options) {
		opts.FilterFunc = f
	}
}

// WithErrorHandler 设置错误处理
func WithErrorHandler(errorHandler ErrorHandler) Option {
	return func(opts *Options) {
		opts.ErrorHandler = errorHandler
	}
}

// WithHeaderMatcher 设置请求头映射
func WithHeaderMatcher(m HeaderMatcher) Option {
	return func(opts *Options) {
		opts.HeaderMatcher = m
	}
}

// WithResponseHandler 设置自定义回包处理
func WithResponseHandler(h CustomResponseHandler) Option {
	return func(opts *Options) {
		opts.ResponseHandler = h
	}
}

// WithFastHTTPErrorHandler 设置 fasthttp 错误处理
func WithFastHTTPErrorHandler(errHandler FastHTTPErrorHandler) Option {
	return func(opts *Options) {
		opts.FastHTTPErrHandler = errHandler
	}
}

// WithFastHTTPHeaderMatcher 设置 fasthttp 请求头映射
func WithFastHTTPHeaderMatcher(m FastHTTPHeaderMatcher) Option {
	return func(opts *Options) {
		opts.FastHTTPHeaderMatcher = m
	}
}

// WithFastHTTPRespHandler 设置 fasthttp 自定义回包处理
func WithFastHTTPRespHandler(h FastHTTPRespHandler) Option {
	return func(opts *Options) {
		opts.FastHTTPRespHandler = h
	}
}

// WithDiscardUnknownParams 设置 忽略未知参数的 query params
func WithDiscardUnknownParams(i bool) Option {
	return func(opts *Options) {
		opts.DiscardUnknownParams = i
	}
}
