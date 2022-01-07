package transport

// ClientTransportOptions client transport可选参数
type ClientTransportOptions struct {
	UDPRecvSize      int
	TCPRecvQueueSize int
}

// ClientTransportOption client transport option function helper
type ClientTransportOption func(*ClientTransportOptions)

// WithClientUDPRecvSize 设置客户端UDP收包大小
func WithClientUDPRecvSize(size int) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.UDPRecvSize = size
	}
}

// WithClientTCPRecvQueueSize 设置客户端TCP缓冲区队列大小
func WithClientTCPRecvQueueSize(size int) ClientTransportOption {
	return func(opts *ClientTransportOptions) {
		opts.TCPRecvQueueSize = size
	}
}

func defaultClientTransportOptions() *ClientTransportOptions {
	return &ClientTransportOptions{UDPRecvSize: 65535}
}
