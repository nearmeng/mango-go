// Package transport 底层网络通讯层，只负责最基本的二进制数据网络通信，没有任何业务逻辑，
// 默认全局只会有一个ServerTransport和一个ClientTransport，提供默认式可插拔能力。
package transport

import (
	"context"
	"reflect"
	"sync"
)

// defaultClientRecvQueueSize 默认的客户端接收大小
var defaultClientRecvQueueSize int = 100000

var (
	clientStreamTrans    = make(map[string]ClientStreamTransport)
	muxClientStreamTrans = sync.RWMutex{}

	serverStreamTrans    = make(map[string]ServerStreamTransport)
	muxServerStreamTrans = sync.RWMutex{}
)

// ClientStreamTransport 用来流式客户端传输，兼容一应一答的transport
type ClientStreamTransport interface {
	// ClientTransport 兼容一应一答的接口，实现Roundtrip方法
	ClientTransport
	// Send 发送流式消息
	Send(ctx context.Context, req []byte, opts ...RoundTripOption) error
	// Recv 接收流式消息
	Recv(ctx context.Context, opts ...RoundTripOption) ([]byte, error)
	// Init 初始化单次流的接口
	Init(ctx context.Context, opts ...RoundTripOption) error
	// Close 关闭流式transport，返回连接到资源池
	Close(ctx context.Context)
}

// ServerStreamTransport  服务端流式传输接口，兼容一应一答的服务端transport
type ServerStreamTransport interface {
	// ServerTransport 兼容老的transport
	ServerTransport
	// Send 发送消息
	Send(ctx context.Context, req []byte) error
	// Close 服务端异常时候调用Close进行现场清理
	Close(ctx context.Context)
}

// RegisterServerStreamTransport 注册流式服务端transport，用户需要实现相应的逻辑
func RegisterServerStreamTransport(name string, t ServerStreamTransport) {
	tv := reflect.ValueOf(t)
	if t == nil || tv.Kind() == reflect.Ptr && tv.IsNil() {
		panic("transport: register nil server transport")
	}
	if name == "" {
		panic("transport: register empty name of server transport")
	}
	muxServerStreamTrans.Lock()
	serverStreamTrans[name] = t
	muxServerStreamTrans.Unlock()

}

// RegisterClientStreamTransport 注册client端的transport插件
func RegisterClientStreamTransport(name string, t ClientStreamTransport) {
	tv := reflect.ValueOf(t)
	if t == nil || tv.Kind() == reflect.Ptr && tv.IsNil() {
		panic("transport: register nil client transport")
	}
	if name == "" {
		panic("transport: register empty name of client transport")
	}
	muxClientStreamTrans.Lock()
	clientStreamTrans[name] = t
	muxClientStreamTrans.Unlock()
}

// GetClientStreamTransport 根据name获取相应的ClientStreamTransport
func GetClientStreamTransport(name string) ClientStreamTransport {
	muxClientStreamTrans.RLock()
	t := clientStreamTrans[name]
	muxClientStreamTrans.RUnlock()
	return t
}

// GetServerStreamTransport 根据name获取相应的ServerStreamTransport
func GetServerStreamTransport(name string) ServerStreamTransport {
	muxServerStreamTrans.RLock()
	t := serverStreamTrans[name]
	muxServerStreamTrans.RUnlock()
	return t
}
