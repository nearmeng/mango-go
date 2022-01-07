package multiplexed

import "time"

// PoolOptions 连接池的一些设置
type PoolOptions struct {
	connectNumber int           // 设置每个地址的连接数
	queueSize     int           // 设置每个连接请求队列长度
	dropFull      bool          // 队列满是否丢弃
	dialTimeout   time.Duration // 连接超时，默认 1s
}

// PoolOption Options helper
type PoolOption func(*PoolOptions)

// WithConnectNumber 设置连接池中，每个对端的连接数
func WithConnectNumber(number int) PoolOption {
	return func(opts *PoolOptions) {
		opts.connectNumber = number
	}
}

// WithQueueSize 设置连接池中，每个连接请求队列长度
func WithQueueSize(number int) PoolOption {
	return func(opts *PoolOptions) {
		opts.queueSize = number
	}
}

// WithDropFull 当队列满时候，是否丢弃请求
func WithDropFull(drop bool) PoolOption {
	return func(opts *PoolOptions) {
		opts.dropFull = drop
	}
}

// WithDialTimeout 设置连接超时
func WithDialTimeout(d time.Duration) PoolOption {
	return func(opts *PoolOptions) {
		opts.dialTimeout = d
	}
}
