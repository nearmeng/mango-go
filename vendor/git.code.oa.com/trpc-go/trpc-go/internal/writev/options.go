package writev

// Options Buffer配置
type Options struct {
	handler    QuitHandler // 设置协程退出清理函数
	bufferSize int         // 设置每个连接请求队列长度
	dropFull   bool        // 队列满是否丢弃
}

// Option 可选参数
type Option func(*Options)

// WithQuitHandler 设置Buffer协程退出处理函数
func WithQuitHandler(handler QuitHandler) Option {
	return func(o *Options) {
		o.handler = handler
	}
}

// WithBufferSize 每个连接请求队列长度
func WithBufferSize(size int) Option {
	return func(opts *Options) {
		opts.bufferSize = size
	}
}

// WithDropFull 当队列满时候，是否丢弃请求
func WithDropFull(drop bool) Option {
	return func(opts *Options) {
		opts.dropFull = drop
	}
}
