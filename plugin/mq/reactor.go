package mq

import (
	"context"

	"github.com/nearmeng/mango-go/plugin/log"
)

// MessageHandler reactor模式中处理消息的handler处理接口
type MessageHandler interface {
	Handle(ctx ReactorContext, msg Message)
}

// Reactor
// @Description: reactor模式的消费者接口
type Reactor interface {
	Register(ctx context.Context, topic []string, handler MessageHandler)
	Run(context.Context)
	Close(context.Context)
}

// ReactorContext reactor模式下的context
// @Description:
type ReactorContext interface {
	context.Context
	Ack(messages ...Message)
}

// ReactorCreator 开发者接入框架需要实现的接口
type ReactorCreator interface {
	NewReactor(context.Context, *ReactorConfig) (Reactor, error)
}

// DefaultHandle 兜底函数，如果用户没有指定消息的处理函数，可用该函数兜底
func DefaultHandle(ctx context.Context, msg *Message) {
	log.Info("using default message handler, %v", msg)
}
