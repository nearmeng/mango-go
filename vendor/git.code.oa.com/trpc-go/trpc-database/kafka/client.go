// Package kafka 封装第三方库sarama
package kafka

import (
	"context"
	"fmt"

	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"github.com/Shopify/sarama"
)

// Client kafka接口
type Client interface {
	Produce(ctx context.Context, key, value []byte,
		headers ...sarama.RecordHeader) error
	SendMessage(ctx context.Context, topic string, key, value []byte,
		headers ...sarama.RecordHeader) (partition int32, offset int64, err error)
	AsyncSendMessage(ctx context.Context, topic string, key, value []byte,
		headers ...sarama.RecordHeader) (err error)
}

// kafkaCli 后端请求结构体
type kafkaCli struct {
	ServiceName string
	Client      client.Client
	opts        []client.Option
}

// NewClientProxy 新建一个kafka后端请求代理 必传参数 kafka服务名: trpc.kafka.producer.service
var NewClientProxy = func(name string, opts ...client.Option) Client {
	c := &kafkaCli{
		ServiceName: name,
		Client:      client.DefaultClient,
	}

	c.opts = make([]client.Option, 0, len(opts)+2)
	c.opts = append(c.opts, client.WithProtocol("kafka"), client.WithDisableServiceRouter())
	c.opts = append(c.opts, opts...)
	return c
}

// Request kafka request body
type Request struct {
	Topic   string
	Key     []byte
	Value   []byte
	Async   bool // 是否异步生产
	Headers []sarama.RecordHeader
}

// Response kafka response body
type Response struct {
	Partition int32
	Offset    int64
}

// Produce 默认同步生产，返回是否发送成功, 可配置async=1 改成异步
func (c *kafkaCli) Produce(ctx context.Context, key, value []byte, headers ...sarama.RecordHeader) error {
	req := &Request{
		Key:   key,
		Value: value,
	}

	if len(headers) > 0 {
		req.Headers = headers
	}

	ctx, msg := codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(fmt.Sprintf("/%s/produce", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // 不序列化
	msg.WithCompressType(0)       // 不压缩
	msg.WithClientReqHead(req)
	rsp, ok := msg.ClientRspHead().(*Response)
	if !ok {
		rsp = &Response{}
		msg.WithClientRspHead(rsp) // 一般用户并不关心offset，只关心是否发送成功，需要offset的数据 可以设置rsphead回传
	}

	return c.Client.Invoke(ctx, req, rsp, c.opts...)
}

// SendMessage 同步生产，返回数据的分区和offset值
func (c *kafkaCli) SendMessage(ctx context.Context, topic string, key, value []byte,
	headers ...sarama.RecordHeader) (partition int32, offset int64, err error) {

	req := &Request{
		Topic: topic,
		Key:   key,
		Value: value,
	}
	if len(headers) > 0 {
		req.Headers = headers
	}
	rsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(fmt.Sprintf("/%s/send", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // 不序列化
	msg.WithCompressType(0)       // 不压缩
	msg.WithClientReqHead(req)
	msg.WithClientRspHead(rsp)

	err = c.Client.Invoke(ctx, req, rsp, c.opts...)
	if err != nil {
		return 0, 0, err
	}

	return rsp.Partition, rsp.Offset, nil
}

// AsyncSendMessage 异步生产，调用方需关注success和error channel捕获消息后记录的本地日志。考虑后期传入回调函数捕获处理这些信息
func (c *kafkaCli) AsyncSendMessage(ctx context.Context, topic string, key, value []byte,
	headers ...sarama.RecordHeader) (err error) {

	req := &Request{
		Topic: topic,
		Key:   key,
		Value: value,
		Async: true,
	}
	if len(headers) > 0 {
		req.Headers = headers
	}
	rsp := &Response{}

	ctx, msg := codec.WithCloneMessage(ctx)
	msg.WithClientRPCName(fmt.Sprintf("/%s/asyncSend", c.ServiceName))
	msg.WithCalleeServiceName(c.ServiceName)
	msg.WithSerializationType(-1) // 不序列化
	msg.WithCompressType(0)       // 不压缩
	msg.WithClientReqHead(req)
	msg.WithClientRspHead(rsp)

	err = c.Client.Invoke(ctx, req, rsp, c.opts...)
	if err != nil {
		return err
	}

	return nil
}
