package codec

import (
	"context"
	"net"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/errs"
)

// ContextKey trpc context key type，context value取值是通过接口判断的，接口会同时判断值和类型，定义新type可以防止重复字符串导致冲突覆盖问题
type ContextKey string

// MetaData 请求message透传字段信息
type MetaData map[string][]byte

var msgPool = sync.Pool{
	New: func() interface{} {
		return &msg{}
	},
}

// Clone 复制一个新的metadata
func (m MetaData) Clone() MetaData {
	md := MetaData{}
	for k, v := range m {
		md[k] = v
	}
	return md
}

// CommonMeta 通用的Meta信息
type CommonMeta map[interface{}]interface{}

// Clone 复制一个新的CommonMeta
func (c CommonMeta) Clone() CommonMeta {
	cm := CommonMeta{}
	for k, v := range c {
		cm[k] = v
	}
	return cm
}

// trpc context key data trpc基本数据
const (
	ContextKeyMessage    = ContextKey("TRPC_MESSAGE")
	ServiceSectionLength = 4 // 点分4段式的trpc服务名称trpc.app.server.service
)

// Msg 多协议通用的消息数据 业务协议打解包的时候需要设置message内部信息
type Msg interface {
	Context() context.Context

	WithRemoteAddr(addr net.Addr) // 设置server端的上游地址，或者client端的下游地址
	WithLocalAddr(addr net.Addr)  // 设置server transport 设置本地地址
	RemoteAddr() net.Addr         // 返回server端的上游地址，或者client端的下游地址
	LocalAddr() net.Addr          // 返回server端的本地地址
	WithNamespace(string)         // 设置server 的 namespace
	Namespace() string            // 返回server的namespace

	WithEnvName(string) // 设置server 的环境
	EnvName() string    // 返回 server的环境

	WithSetName(string) // 设置server所在的set
	SetName() string    // 返回server所在的set

	WithEnvTransfer(string) // 设置服务透传的环境信息
	EnvTransfer() string    // 返回透传的环境信息

	WithRequestTimeout(time.Duration) // 设置server codec 设置上游超时，client设置下游超时
	RequestTimeout() time.Duration    // 返回server 设置的上游超时，或者client设置的下游超时

	WithSerializationType(int) // 设置Serialization类型
	SerializationType() int    // 返回Serialization类型

	WithCompressType(int) // 设置压缩类型
	CompressType() int    // 返回压缩类型

	WithServerRPCName(string) // 设置server codec 设置当前server handler调用方法名
	WithClientRPCName(string) // client stub 设置下游调用方法名

	ServerRPCName() string // 返回当前server handler调用方法名：/trpc.app.server.service/method
	ClientRPCName() string // 返回调用下游的接口方法名

	WithCallerServiceName(string) // 设置调用方服务名
	WithCalleeServiceName(string) // 设置被调方服务名

	WithCallerApp(string)     // 设置主调app server角度是上游的app，client角度是自身的app
	WithCallerServer(string)  // 设置主调server server角度是上游的server，client角度是自身的server
	WithCallerService(string) // 设置主调service server角度是上游的service，client角度是自身的service
	WithCallerMethod(string)  // 设置主调method server角度是上游的method，client角度是自身的method

	WithCalleeApp(string)     // 设置被调app server角度是自身app，client角度是下游的app
	WithCalleeServer(string)  // 设置被调server server角度是自身server，client角度是下游的server
	WithCalleeService(string) // 设置被调service server角度是自身service，client角度是下游的service
	WithCalleeMethod(string)  // 设置被调method server角度是自身method，client角度是下游的method

	CallerServiceName() string // 返回主调服务名：trpc.app.server.service server角度是上游服务名，client角度是自身的服务名
	CallerApp() string         // 返回主调app server角度是上游的app，client角度是自身的app
	CallerServer() string      // 返回主调server server角度是上游的server，client角度是自身的server
	CallerService() string     // 返回主调service server角度是上游的service，client角度是自身的service
	CallerMethod() string      // 返回主调method server角度是上游的method，client角度是自身的method

	CalleeServiceName() string // 返回被调服务名 server角度是自身服务名，client角度是下游的服务名
	CalleeApp() string         // 返回被调app server角度是自身app，client角度是下游的app
	CalleeServer() string      // 返回被调server server角度是自身server，client角度是下游的server
	CalleeService() string     // 返回被调service server角度是自身service，client角度是下游的service
	CalleeMethod() string      // 返回被调method server角度是自身method，client角度是下游的method

	CalleeContainerName() string    // 设置被调服务容器名
	WithCalleeContainerName(string) // 返回被调服务容器名

	WithServerMetaData(MetaData) // 设置服务端收到请求的协议透传字段
	ServerMetaData() MetaData    // 返回服务端收到请求的协议透传字段

	WithFrameHead(interface{}) // 设置帧头
	FrameHead() interface{}    // 返回帧头

	WithServerReqHead(interface{}) // 设置服务端收到请求的协议包头
	ServerReqHead() interface{}    // 返回服务端收到请求的协议包头

	WithServerRspHead(interface{}) // 设置服务端回给上游的协议包头
	ServerRspHead() interface{}    // 返回服务端回给上游的协议包头

	WithDyeing(bool) // 设置染色标记
	Dyeing() bool    // 返回染色标记

	WithDyeingKey(string) // 设置染色key
	DyeingKey() string    // 返回染色key

	WithServerRspErr(error)    // 设置服务端回包时设置的错误
	ServerRspErr() *errs.Error // 返回服务端回包时设置的错误

	WithClientMetaData(MetaData) // 设置调用下游时的客户端协议透传字段
	ClientMetaData() MetaData    // 返回调用下游时的客户端协议透传字段

	WithClientReqHead(interface{}) // 设置客户端请求下游设置的协议包头
	ClientReqHead() interface{}    // 返回客户端请求下游设置的协议包头

	WithClientRspErr(error) // 设置客户端调用下游时返回的err
	ClientRspErr() error    // 返回客户端调用下游时返回的err

	WithClientRspHead(interface{}) // 设置下游回包的协议包头
	ClientRspHead() interface{}    // 返回下游回包的协议包头

	WithLogger(interface{}) // 设置日志logger到context msg里面
	Logger() interface{}    // 返回日志logger到context msg里面

	WithRequestID(uint32) // 设置request id
	RequestID() uint32    // 返回request id
	WithStreamID(uint32)  // 设置流ID
	StreamID() uint32     // 返回流ID

	StreamFrame() interface{}    // 设置流式帧
	WithStreamFrame(interface{}) // 返回流式帧

	WithCalleeSetName(string) // 设置被调的set名字
	CalleeSetName() string    // 返回被调的set名字

	WithCommonMeta(CommonMeta) // 设置通用的Metadata,携带额外的元数据信息
	CommonMeta() CommonMeta    // 返回通用的Metadata，携带额外的元数据信息

	WithCallType(RequestType) // 设置请求类型
	CallType() RequestType    // 返回请求类型
}

// CopyMsg copy src Msg to dst.
// All fields of src msg will be copied to dst msg.
func CopyMsg(dst, src Msg) {
	if dst == nil || src == nil {
		return
	}
	dst.WithFrameHead(src.FrameHead())
	dst.WithRequestTimeout(src.RequestTimeout())
	dst.WithSerializationType(src.SerializationType())
	dst.WithCompressType(src.CompressType())
	dst.WithStreamID(src.StreamID())
	dst.WithDyeing(src.Dyeing())
	dst.WithDyeingKey(src.DyeingKey())
	dst.WithServerRPCName(src.ServerRPCName())
	dst.WithClientRPCName(src.ClientRPCName())
	dst.WithServerMetaData(src.ServerMetaData().Clone())
	dst.WithClientMetaData(src.ClientMetaData().Clone())
	dst.WithCallerServiceName(src.CallerServiceName())
	dst.WithCalleeServiceName(src.CalleeServiceName())
	dst.WithCalleeContainerName(src.CalleeContainerName())
	dst.WithServerRspErr(src.ServerRspErr())
	dst.WithClientRspErr(src.ClientRspErr())
	dst.WithServerReqHead(src.ServerReqHead())
	dst.WithServerRspHead(src.ServerRspHead())
	dst.WithClientReqHead(src.ClientReqHead())
	dst.WithClientRspHead(src.ClientRspHead())
	dst.WithLocalAddr(src.LocalAddr())
	dst.WithRemoteAddr(src.RemoteAddr())
	dst.WithLogger(src.Logger())
	dst.WithCallerApp(src.CallerApp())
	dst.WithCallerServer(src.CallerServer())
	dst.WithCallerService(src.CallerService())
	dst.WithCallerMethod(src.CallerMethod())
	dst.WithCalleeApp(src.CalleeApp())
	dst.WithCalleeServer(src.CalleeServer())
	dst.WithCalleeService(src.CalleeService())
	dst.WithCalleeMethod(src.CalleeMethod())
	dst.WithNamespace(src.Namespace())
	dst.WithSetName(src.SetName())
	dst.WithEnvName(src.EnvName())
	dst.WithEnvTransfer(src.EnvTransfer())
	dst.WithRequestID(src.RequestID())
	dst.WithStreamFrame(src.StreamFrame())
	dst.WithCalleeSetName(src.CalleeSetName())
	dst.WithCommonMeta(src.CommonMeta().Clone())
	dst.WithCallType(src.CallType())
}
