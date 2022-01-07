package kafka

import (
	"context"
	"fmt"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/server"
	"github.com/Shopify/sarama"
)

// Consumer 消费者
type Consumer interface {
	//接收到消息时的回调函数
	Handle(ctx context.Context, key, value []byte, topic string, partition int32, offset int64) error
}

// Consumer_Handle consumer service handler wrapper
func Consumer_Handle(svr interface{}, ctx context.Context, f server.FilterFunc) (rspbody interface{}, err error) {
	filters, err := f(nil)
	if err != nil {
		return nil, err
	}

	handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
		msg := codec.Message(ctx)
		m, ok := msg.ServerReqHead().(*sarama.ConsumerMessage)
		if !ok {
			return errs.NewFrameError(errs.RetServerDecodeFail, "kafka consumer handler: message type invalid")
		}
		return svr.(Consumer).Handle(ctx, m.Key, m.Value, m.Topic, m.Partition, m.Offset)
	}

	err = filters.Handle(ctx, nil, nil, handleFunc)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ConsumerService_ServiceDesc descriptor for server.RegisterService
var ConsumerService_ServiceDesc = server.ServiceDesc{
	ServiceName: fmt.Sprintf("trpc.kafka.consumer.service"),
	HandlerType: ((*Consumer)(nil)),
	Methods: []server.Method{{
		Name: "/trpc.kafka.consumer.service/handle",
		Func: Consumer_Handle,
	}},
}

// RegisterConsumerService register consumer service
func RegisterConsumerService(s server.Service, svr Consumer) {
	s.Register(&ConsumerService_ServiceDesc, svr)
}

type handler func(ctx context.Context, key, value []byte, topic string, partition int32, offset int64) error

// Handle 主处理
func (h handler) Handle(ctx context.Context, key, value []byte, topic string, partition int32, offset int64) error {
	return h(ctx, key, value, topic, partition, offset)
}

// RegisterHandlerService register consumer function
func RegisterHandlerService(s server.Service, handle func(ctx context.Context, key, value []byte,
	topic string, partition int32, offset int64) error) {
	s.Register(&ConsumerService_ServiceDesc, handler(handle))
}

// BatchConsumer 批量消费者
type BatchConsumer interface {
	//接收到消息时的回调函数
	Handle(ctx context.Context, msgArray []*sarama.ConsumerMessage) error
}

type batchHandler func(ctx context.Context, msgArray []*sarama.ConsumerMessage) error

// Handle handle
func (h batchHandler) Handle(ctx context.Context, msgArray []*sarama.ConsumerMessage) error {
	return h(ctx, msgArray)
}

// BatchConsumerServiceDesc descriptor for server.RegisterService
var BatchConsumerServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.kafka.consumer.service",
	HandlerType: ((*BatchConsumer)(nil)),
	Methods: []server.Method{
		{
			Name: "/trpc.kafka.consumer.service/handle",
			Func: BatchConsumerHandle,
		},
	},
}

// BatchConsumerHandle batch consumer service handler wrapper
func BatchConsumerHandle(svr interface{}, ctx context.Context, f server.FilterFunc) (rspbody interface{}, err error) {
	filters, err := f(nil)
	if err != nil {
		return nil, err
	}

	handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
		msg := codec.Message(ctx)
		msgs, ok := msg.ServerReqHead().([]*sarama.ConsumerMessage)
		if ok {
			return svr.(BatchConsumer).Handle(ctx, msgs)
		}
		return errs.NewFrameError(errs.RetServerDecodeFail, "kafka consumer handler: message type invalid")
	}

	err = filters.Handle(ctx, nil, nil, handleFunc)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// RegisterBatchHandlerService register consumer function
func RegisterBatchHandlerService(s server.Service,
	handle func(ctx context.Context, msgArray []*sarama.ConsumerMessage) error) {
	_ = s.Register(&BatchConsumerServiceDesc, batchHandler(handle))
}
