package mq

import (
	"context"
)

// Client
// @Description: mq 连接客户端抽象
type Client interface {
	NewReader(ctx context.Context, config *ReaderConfig) (Reader, error)
	NewWriter(ctx context.Context, config *WriterConfig) (Writer, error)
	TopicPartitions(topic string) ([]string, error)
	ResourceManager() ResourceManager

	Close()
}

// Writer
// @Description:  mq 生产者接口
type Writer interface {
	GetConfig() *WriterConfig
	SetConfig(config *WriterConfig)
	WriteMessage(ctx context.Context, msg Message) (int64, error)
	WriteMessageAsync(ctx context.Context, msg Message, backFunc CallBackFunc) error
	AddWriterInterceptor(interceptor WriterInterceptor)
	RemoveWriterInterceptor(interceptor WriterInterceptor)
	Close()
}

// QueueMetaOption
// @Description: mq 队列原始信息接口
type QueueMetaOption interface {
	QueueName() string
	Partition() int
}

// Reader
// @Description: mq消费者接口
type Reader interface {
	GetConfig() *ReaderConfig
	SetConfig(config *ReaderConfig)
	ReadMessage(ctx context.Context) (Message, error)
	ReadMessageWithOption(ctx context.Context, info QueueMetaOption) (Message, error)
	AddReaderInterceptor(interceptor ReaderInterceptor)
	RemoveReaderInterceptor(interceptor ReaderInterceptor)
	Ack(ctx context.Context, messages ...Message)
	Close()
}

// CallBackFunc 异步发送后调用的回调函数
type CallBackFunc func(seqID int64, message Message, err error)
