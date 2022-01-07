package rollwriter

// AsyncOptions AsyncRollWriter类调用参数
type AsyncOptions struct {
	// LogQueueSize 日志异步队列大小
	LogQueueSize int

	// WriteLogSize 触发日志异步写入的大小阈值
	WriteLogSize int

	// WriteLogInterval 触发日志异步写入的时间隔阈值
	WriteLogInterval int

	// DropLog 是否丢弃日志（当日志异步队列写满时）
	DropLog bool
}

// AsyncOption 调用参数工具函数
type AsyncOption func(*AsyncOptions)

// WithLogQueueSize 设置日志异步队列大小
func WithLogQueueSize(n int) AsyncOption {
	return func(o *AsyncOptions) {
		o.LogQueueSize = n
	}
}

// WithWriteLogSize 设置触发日志异步写入的大小阈值(单位字节)
func WithWriteLogSize(n int) AsyncOption {
	return func(o *AsyncOptions) {
		o.WriteLogSize = n
	}
}

// WithWriteLogInterval 设置触发日志异步写入的时间隔阈值(单位ms)
func WithWriteLogInterval(n int) AsyncOption {
	return func(o *AsyncOptions) {
		o.WriteLogInterval = n
	}
}

// WithDropLog 设置是否丢弃日志（当日志异步队列写满时）
func WithDropLog(b bool) AsyncOption {
	return func(o *AsyncOptions) {
		o.DropLog = b
	}
}
