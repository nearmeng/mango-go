package kafka

import (
	"fmt"
	"os"
	"path"

	"git.code.oa.com/trpc-go/trpc-go/codec"
)

func init() {
	codec.Register("kafka", DefaultServerCodec, DefaultClientCodec)
}

// default codec
var (
	DefaultServerCodec = &ServerCodec{}
	DefaultClientCodec = &ClientCodec{}

	serverName = path.Base(os.Args[0])
)

// ServerCodec timer服务端编解码
type ServerCodec struct {
}

// Decode 服务端收到客户端二进制请求数据解包到reqbody, service handler会自动创建一个新的空的msg 作为初始通用消息体
func (s *ServerCodec) Decode(kafkaMsg codec.Msg, reqbuf []byte) (kafkareqbody []byte, err error) {
	//设置上游服务名
	kafkaMsg.WithCallerServiceName("trpc.kafka.noserver.noservice")

	kafkaMsg.WithServerRPCName("/trpc.kafka.consumer.service/handle")

	kafkaMsg.WithCalleeApp("kafka")
	kafkaMsg.WithCalleeServer(serverName)
	kafkaMsg.WithCalleeService("service")

	return kafkareqbody, nil
}

// Encode 服务端打包rspbody到二进制 回给客户端
func (s *ServerCodec) Encode(kafkaMsg codec.Msg, rspbody []byte) (rspbuf []byte, err error) {
	return nil, nil
}

// ClientCodec 解码kafka client请求
type ClientCodec struct{}

// Encode 设置kafka client请求的元数据
func (c *ClientCodec) Encode(kafkaMsg codec.Msg, _ []byte) ([]byte, error) {
	//自身
	if kafkaMsg.CallerServiceName() == "" {
		kafkaMsg.WithCallerServiceName(fmt.Sprintf("trpc.kafka.%s.service", serverName))
	}
	return nil, nil
}

// Decode 解析kafka client回包里的元数据
func (c *ClientCodec) Decode(kafkaMsg codec.Msg, _ []byte) ([]byte, error) {
	return nil, nil
}
