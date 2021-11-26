package mq

import (
	"context"
)


// ReaderInterceptor
// @Description: 消费者拦截器接口，在调用read前后执行
type ReaderInterceptor interface {
	PreRead(context.Context, Message)
	AfterRead(context.Context, Message)
}

// WriterInterceptor
// @Description: 生产者者拦截器接口 在调用write前后执行
type WriterInterceptor interface {
	PreWrite(context.Context, Message)
	AfterWrite(context.Context, Message)
}
