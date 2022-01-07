package codec

import (
	"context"
	"net"
	"strings"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/errs"
)

// msg rpc上下文信息
type msg struct {
	context             context.Context
	frameHead           interface{}
	requestTimeout      time.Duration
	serializationType   int
	compressType        int
	streamID            uint32
	dyeing              bool
	dyeingKey           string
	serverRPCName       string
	clientRPCName       string
	serverMetaData      MetaData
	clientMetaData      MetaData
	callerServiceName   string
	calleeServiceName   string
	calleeContainerName string
	serverRspErr        error
	clientRspErr        error
	serverReqHead       interface{}
	serverRspHead       interface{}
	clientReqHead       interface{}
	clientRspHead       interface{}
	localAddr           net.Addr
	remoteAddr          net.Addr
	logger              interface{}
	callerApp           string
	callerServer        string
	callerService       string
	callerMethod        string
	calleeApp           string
	calleeServer        string
	calleeService       string
	calleeMethod        string
	namespace           string
	setName             string
	envName             string
	envTransfer         string
	requestID           uint32
	calleeSetName       string
	streamFrame         interface{}
	commonMeta          CommonMeta
	callType            RequestType
}

// resetDefault 将msg的所有成员变量变成默认值
func (m *msg) resetDefault() {
	m.context = nil
	m.frameHead = nil
	m.requestTimeout = 0
	m.serializationType = 0
	m.compressType = 0
	m.dyeing = false
	m.dyeingKey = ""
	m.serverRPCName = ""
	m.clientRPCName = ""
	m.serverMetaData = nil
	m.clientMetaData = nil
	m.callerServiceName = ""
	m.calleeServiceName = ""
	m.calleeContainerName = ""
	m.serverRspErr = nil
	m.clientRspErr = nil
	m.serverReqHead = nil
	m.serverRspHead = nil
	m.clientReqHead = nil
	m.clientRspHead = nil
	m.localAddr = nil
	m.remoteAddr = nil
	m.logger = nil
	m.callerApp = ""
	m.callerServer = ""
	m.callerService = ""
	m.callerMethod = ""
	m.calleeApp = ""
	m.calleeServer = ""
	m.calleeService = ""
	m.calleeMethod = ""
	m.namespace = ""
	m.setName = ""
	m.envName = ""
	m.envTransfer = ""
	m.requestID = 0
	m.streamFrame = nil
	m.streamID = 0
	m.calleeSetName = ""
	m.commonMeta = nil
	m.callType = 0
}

// Context 新建msg时，保存老的ctx
func (m *msg) Context() context.Context {
	return m.context
}

// WithNamespace 设置 server 的 namespace
func (m *msg) WithNamespace(namespace string) {
	m.namespace = namespace
}

// Namespace 返回 namespace
func (m *msg) Namespace() string {
	return m.namespace
}

// WithEnvName 设置环境
func (m *msg) WithEnvName(envName string) {
	m.envName = envName
}

// WithSetName 设置set分组
func (m *msg) WithSetName(setName string) {
	m.setName = setName
}

// SetName 返回set分组
func (m *msg) SetName() string {
	return m.setName
}

// WithCalleeSetName 设置被调的set分组
func (m *msg) WithCalleeSetName(s string) {
	m.calleeSetName = s
}

// CalleeSetName 返回被调的set分组
func (m *msg) CalleeSetName() string {
	return m.calleeSetName
}

// EnvName 返回环境
func (m *msg) EnvName() string {
	return m.envName
}

// WithEnvTransfer 设置透传环境信息
func (m *msg) WithEnvTransfer(envTransfer string) {
	m.envTransfer = envTransfer
}

// EnvTransfer 返回透传环境信息
func (m *msg) EnvTransfer() string {
	return m.envTransfer
}

// WithRemoteAddr 设置 remoteAddr
func (m *msg) WithRemoteAddr(addr net.Addr) {
	m.remoteAddr = addr
}

// WithLocalAddr 设置 localAddr
func (m *msg) WithLocalAddr(addr net.Addr) {
	m.localAddr = addr
}

// RemoteAddr 获取 remoteAddr
func (m *msg) RemoteAddr() net.Addr {
	return m.remoteAddr
}

// LocalAddr 获取 localAddr
func (m *msg) LocalAddr() net.Addr {
	return m.localAddr
}

// RequestTimeout 上游业务协议里面设置的请求超时时间
func (m *msg) RequestTimeout() time.Duration {
	return m.requestTimeout
}

// WithRequestTimeout 设置请求超时时间
func (m *msg) WithRequestTimeout(t time.Duration) {
	m.requestTimeout = t
}

// FrameHead 返回桢信息
func (m *msg) FrameHead() interface{} {
	return m.frameHead
}

// WithFrameHead 设置桢信息
func (m *msg) WithFrameHead(f interface{}) {
	m.frameHead = f
}

// SerializationType body序列化方式 见 serialization.go里面的常量定义
func (m *msg) SerializationType() int {
	return m.serializationType
}

// WithSerializationType 设置body序列化方式
func (m *msg) WithSerializationType(t int) {
	m.serializationType = t
}

// CompressType body解压缩方式 见 compress.go里面的常量定义
func (m *msg) CompressType() int {
	return m.compressType
}

// WithCompressType 设置body解压缩方式
func (m *msg) WithCompressType(t int) {
	m.compressType = t
}

// ServerRPCName 服务端接收到请求的协议的rpc name
func (m *msg) ServerRPCName() string {
	return m.serverRPCName
}

// WithServerRPCName 设置服务端rpc name
func (m *msg) WithServerRPCName(s string) {
	if m.serverRPCName == s {
		return
	}
	m.serverRPCName = s

	if method, ok := getMethodFromRPCName(s); ok {
		m.WithCalleeMethod(method)
	}
}

// ClientRPCName 后端调用设置的rpc name
func (m *msg) ClientRPCName() string {
	return m.clientRPCName
}

// WithClientRPCName 设置后端调用rpc name，由client stub调用
func (m *msg) WithClientRPCName(s string) {
	if m.clientRPCName == s {
		return
	}
	m.clientRPCName = s

	if method, ok := getMethodFromRPCName(s); ok {
		m.WithCalleeMethod(method)
	}
}

// ServerMetaData 服务端收到请求的协议透传字段
func (m *msg) ServerMetaData() MetaData {
	return m.serverMetaData
}

// WithServerMetaData 设置服务端收到请求的协议透传字段
func (m *msg) WithServerMetaData(d MetaData) {
	if d == nil {
		d = MetaData{}
	}
	m.serverMetaData = d
}

// ClientMetaData 调用下游时的客户端协议透传字段
func (m *msg) ClientMetaData() MetaData {
	return m.clientMetaData
}

// WithClientMetaData 设置调用下游时的客户端协议透传字段
func (m *msg) WithClientMetaData(d MetaData) {
	if d == nil {
		d = MetaData{}
	}
	m.clientMetaData = d
}

// CalleeServiceName 被调方的服务名
func (m *msg) CalleeServiceName() string {
	return m.calleeServiceName
}

// WithCalleeServiceName 设置被调方的服务名
func (m *msg) WithCalleeServiceName(s string) {
	if m.calleeServiceName == s {
		return
	}
	m.calleeServiceName = s

	app, server, service, ok := getAppServerService(s)
	if !ok {
		return
	}
	m.WithCalleeApp(app)
	m.WithCalleeServer(server)
	m.WithCalleeService(service)
}

// CalleeContainerName 被调方的容器名
func (m *msg) CalleeContainerName() string {
	return m.calleeContainerName
}

// WithCalleeContainerName 设置被调方的容器名
func (m *msg) WithCalleeContainerName(s string) {
	m.calleeContainerName = s
}

// WithStreamFrame 设置流式帧
func (m *msg) WithStreamFrame(i interface{}) {
	m.streamFrame = i
}

// StreamFrame 返回流式帧
func (m *msg) StreamFrame() interface{} {
	return m.streamFrame
}

// CallerServiceName 主调方的服务名
func (m *msg) CallerServiceName() string {
	return m.callerServiceName
}

// WithCallerServiceName 设置主调方的服务名
func (m *msg) WithCallerServiceName(s string) {
	if m.callerServiceName == s {
		return
	}
	m.callerServiceName = s

	app, server, service, ok := getAppServerService(s)
	if !ok {
		return
	}
	m.WithCallerApp(app)
	m.WithCallerServer(server)
	m.WithCallerService(service)
}

// ServerRspErr 服务端回包时设置的错误，一般为handler返回的err
func (m *msg) ServerRspErr() *errs.Error {
	if m.serverRspErr == nil {
		return nil
	}
	e, ok := m.serverRspErr.(*errs.Error)
	if !ok {
		return &errs.Error{
			Type: errs.ErrorTypeBusiness,
			Code: errs.RetUnknown,
			Msg:  m.serverRspErr.Error(),
		}
	}
	return e
}

// WithServerRspErr 设置服务端回包时设置的错误
func (m *msg) WithServerRspErr(e error) {
	m.serverRspErr = e
}

// WithStreamID 设置流ID
func (m *msg) WithStreamID(streamID uint32) {
	m.streamID = streamID
}

// StreamID 返回流ID
func (m *msg) StreamID() uint32 {
	return m.streamID
}

// ClientRspErr 客户端调用下游时返回的err
func (m *msg) ClientRspErr() error {
	return m.clientRspErr
}

// WithClientRspErr 设置客户端调用下游时返回的err 一般由客户端解析回包时调用
func (m *msg) WithClientRspErr(e error) {
	m.clientRspErr = e
}

// ServerReqHead 服务端收到请求的协议包头
func (m *msg) ServerReqHead() interface{} {
	return m.serverReqHead
}

// WithServerReqHead 设置服务端收到请求的协议包头
func (m *msg) WithServerReqHead(h interface{}) {
	m.serverReqHead = h
}

// ServerRspHead 服务端回给上游的协议包头
func (m *msg) ServerRspHead() interface{} {
	return m.serverRspHead
}

// WithServerRspHead 设置服务端回给上游的协议包头
func (m *msg) WithServerRspHead(h interface{}) {
	m.serverRspHead = h
}

// ClientReqHead 客户端请求下游设置的协议包头，一般不用设置，主要用于跨协议调用
func (m *msg) ClientReqHead() interface{} {
	return m.clientReqHead
}

// WithClientReqHead 设置客户端请求下游设置的协议包头
func (m *msg) WithClientReqHead(h interface{}) {
	m.clientReqHead = h
}

// ClientRspHead 下游回包的协议包头
func (m *msg) ClientRspHead() interface{} {
	return m.clientRspHead
}

// WithClientRspHead 设置下游回包的协议包头
func (m *msg) WithClientRspHead(h interface{}) {
	m.clientRspHead = h
}

// Dyeing 染色标记
func (m *msg) Dyeing() bool {
	return m.dyeing
}

// WithDyeing 设置染色标记
func (m *msg) WithDyeing(dyeing bool) {
	m.dyeing = dyeing
}

// DyeingKey 染色key
func (m *msg) DyeingKey() string {
	return m.dyeingKey
}

// WithDyeingKey 设置染色key
func (m *msg) WithDyeingKey(key string) {
	m.dyeingKey = key
}

// CallerApp 返回主调app
func (m *msg) CallerApp() string {
	return m.callerApp
}

// WithCallerApp 设置主调app
func (m *msg) WithCallerApp(app string) {
	m.callerApp = app
}

// CallerServer 返回主调server
func (m *msg) CallerServer() string {
	return m.callerServer
}

// WithCallerServer 设置主调app
func (m *msg) WithCallerServer(s string) {
	m.callerServer = s
}

// CallerService 返回主调service
func (m *msg) CallerService() string {
	return m.callerService
}

// WithCallerService 设置主调service
func (m *msg) WithCallerService(s string) {
	m.callerService = s
}

// WithCallerMethod 设置主调method
func (m *msg) WithCallerMethod(s string) {
	m.callerMethod = s
}

// CallerMehtod 返回主调method
func (m *msg) CallerMethod() string {
	return m.callerMethod
}

// CalleeApp 返回被调app
func (m *msg) CalleeApp() string {
	return m.calleeApp
}

// WithCalleeApp 设置被调app
func (m *msg) WithCalleeApp(app string) {
	m.calleeApp = app
}

// CalleeServer 返回被调server
func (m *msg) CalleeServer() string {
	return m.calleeServer
}

// WithCalleeServer 设置被调server
func (m *msg) WithCalleeServer(s string) {
	m.calleeServer = s
}

// CalleeService 返回被调service
func (m *msg) CalleeService() string {
	return m.calleeService
}

// WithCalleeService 设置被调service
func (m *msg) WithCalleeService(s string) {
	m.calleeService = s
}

// WithCalleeMethod 设置被调调method
func (m *msg) WithCalleeMethod(s string) {
	m.calleeMethod = s
}

// CalleeMehtod 返回被调method
func (m *msg) CalleeMethod() string {
	return m.calleeMethod
}

// WithLogger 设置日志logger到context msg里面，一般设置的是WithFields生成的新的logger
func (m *msg) WithLogger(l interface{}) {
	m.logger = l
}

// Logger 从msg取出logger
func (m *msg) Logger() interface{} {
	return m.logger
}

// WithRequestID 设置request id
func (m *msg) WithRequestID(id uint32) {
	m.requestID = id
}

// RequestID 获取request id
func (m *msg) RequestID() uint32 {
	return m.requestID
}

// WithCommonMeta 设置通用的Meta信息
func (m *msg) WithCommonMeta(c CommonMeta) {
	m.commonMeta = c
}

// CommonMeta 返回通用的Meta信息
func (m *msg) CommonMeta() CommonMeta {
	return m.commonMeta
}

// WithCallType 设置请求类型
func (m *msg) WithCallType(t RequestType) {
	m.callType = t
}

// CallType 返回请求类型
func (m *msg) CallType() RequestType {
	return m.callType
}

// WithNewMessage 创建新的空的message 放到ctx里面 server收到请求入口创建
func WithNewMessage(ctx context.Context) (context.Context, Msg) {

	m := msgPool.Get().(*msg)
	ctx = context.WithValue(ctx, ContextKeyMessage, m)
	m.context = ctx
	return ctx, m
}

// PutBackMessage return struct Message to sync pool,
// and reset all the members of Message to default
func PutBackMessage(sourceMsg Msg) {
	m, ok := sourceMsg.(*msg)
	if !ok {
		return
	}
	m.resetDefault()
	msgPool.Put(m)
}

// WithCloneContextAndMessage 创建新的context， 拷贝当前context里面的message到新的context
// 返回新的context和message，流式使用
func WithCloneContextAndMessage(ctx context.Context) (context.Context, Msg) {

	newMsg := msgPool.Get().(*msg)
	newCtx := context.Background()
	val := ctx.Value(ContextKeyMessage)
	m, ok := val.(*msg)
	if !ok {
		newCtx = context.WithValue(newCtx, ContextKeyMessage, newMsg)
		newMsg.context = newCtx
		return newCtx, newMsg
	}

	newCtx = context.WithValue(newCtx, ContextKeyMessage, newMsg)
	newMsg.context = newCtx

	copyCommonMessage(m, newMsg)
	copyServerToServerMessage(m, newMsg)
	return newCtx, newMsg
}

// copyCommonMessage 拷贝msg的通用数据
func copyCommonMessage(m *msg, newMsg *msg) {
	newMsg.frameHead = m.frameHead
	newMsg.requestTimeout = m.requestTimeout
	newMsg.serializationType = m.serializationType
	newMsg.serverRPCName = m.serverRPCName
	newMsg.clientRPCName = m.clientRPCName
	newMsg.serverReqHead = m.serverReqHead
	newMsg.serverRspHead = m.serverRspHead
	newMsg.dyeing = m.dyeing
	newMsg.dyeingKey = m.dyeingKey
	newMsg.serverMetaData = m.serverMetaData
	newMsg.logger = m.logger
	newMsg.namespace = m.namespace
	newMsg.envName = m.envName
	newMsg.setName = m.setName
	newMsg.envTransfer = m.envTransfer
	newMsg.commonMeta = m.commonMeta.Clone()
}

// copyClientMessage 从服务端透传给客户端的msg
func copyServerToClientMessage(m *msg, newMsg *msg) {
	newMsg.clientMetaData = m.serverMetaData.Clone()
	// clone是给下游client用的，所以caller等于callee
	newMsg.callerServiceName = m.calleeServiceName
	newMsg.callerApp = m.calleeApp
	newMsg.callerServer = m.calleeServer
	newMsg.callerService = m.calleeService
	newMsg.callerMethod = m.calleeMethod
}

func copyServerToServerMessage(m *msg, newMsg *msg) {
	newMsg.callerServiceName = m.callerServiceName
	newMsg.callerApp = m.callerApp
	newMsg.callerServer = m.callerServer
	newMsg.callerService = m.callerService
	newMsg.callerMethod = m.callerMethod

	newMsg.calleeServiceName = m.calleeServiceName
	newMsg.calleeService = m.calleeService
	newMsg.calleeApp = m.calleeApp
	newMsg.calleeServer = m.calleeServer
	newMsg.calleeMethod = m.calleeMethod
}

// WithCloneMessage 复制一个新的message 放到ctx里面 每次rpc调用都必须创建新的msg 由client stub调用该函数
func WithCloneMessage(ctx context.Context) (context.Context, Msg) {
	newMsg := msgPool.Get().(*msg)
	val := ctx.Value(ContextKeyMessage)
	m, ok := val.(*msg)
	if !ok {
		ctx = context.WithValue(ctx, ContextKeyMessage, newMsg)
		newMsg.context = ctx
		return ctx, newMsg
	}
	ctx = context.WithValue(ctx, ContextKeyMessage, newMsg)
	newMsg.context = ctx
	copyCommonMessage(m, newMsg)
	copyServerToClientMessage(m, newMsg)
	return ctx, newMsg
}

// Message 从ctx取出message
func Message(ctx context.Context) Msg {
	val := ctx.Value(ContextKeyMessage)
	m, ok := val.(*msg)
	if !ok {
		return &msg{context: ctx}
	}
	return m
}

// EnsureMessage 确保 ctx 中存在一个 msg。如果 ctx 中设置了 msg，就返回原始 ctx 和它的 msg，否则，返回一个设置了新 msg 的 ctx。
func EnsureMessage(ctx context.Context) (context.Context, Msg) {
	val := ctx.Value(ContextKeyMessage)
	if m, ok := val.(*msg); ok {
		return ctx, m
	}
	return WithNewMessage(ctx)
}

// getAppServerService 从ServiceName字符串中提取app, server, service字段
// ServiceName的格式为：trpc.app.server.service
func getAppServerService(s string) (app, server, service string, ok bool) {
	if strings.Count(s, ".") != (ServiceSectionLength - 1) {
		ok = false
		return
	}
	i := strings.Index(s, ".") + 1
	j := strings.Index(s[i:], ".") + i + 1
	k := strings.Index(s[j:], ".") + j + 1

	app = s[i : j-1]
	server = s[j : k-1]
	service = s[k:]
	ok = true
	return
}

// getMethodFromRPCName 从RPC字符中提取method字段
// RPC字符串格式为：/trpc.app.server.service/method
func getMethodFromRPCName(s string) (string, bool) {
	if strings.Count(s, "/") != 2 {
		return "", false
	}
	i := strings.Index(s, "/") + 1
	j := strings.Index(s[i:], "/") + i + 1
	return s[j:], true
}
