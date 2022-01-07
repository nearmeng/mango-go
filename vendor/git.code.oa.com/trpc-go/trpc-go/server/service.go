package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/overloadctrl"
	"git.code.oa.com/trpc-go/trpc-go/restful"
	"git.code.oa.com/trpc-go/trpc-go/transport"
)

// MaxCloseWaitTime 关闭service最长等待时间
const MaxCloseWaitTime = 10 * time.Second

// Service 服务端调用结构
type Service interface {
	// 注册路由信息
	Register(serviceDesc interface{}, serviceImpl interface{}) error
	// 启动服务
	Serve() error
	// 关闭服务
	Close(chan struct{}) error
}

// FilterFunc 内部解析reqbody 并返回拦截器 传给server stub
type FilterFunc func(reqbody interface{}) (filter.Chain, error)

// Method 服务rpc方法信息
type Method struct {
	Name     string
	Func     func(svr interface{}, ctx context.Context, f FilterFunc) (rspbody interface{}, err error)
	Bindings []*restful.Binding
}

// ServiceDesc 服务描述service定义
type ServiceDesc struct {
	ServiceName  string
	HandlerType  interface{}
	Methods      []Method
	Streams      []StreamDesc
	StreamHandle StreamHandle
}

// StreamDesc 流式的描述
type StreamDesc struct {
	// StreamName 当前流的名字
	StreamName string
	// Handler service的流式处理逻辑
	Handler StreamHandlerWapper
	// ServerStreams描述是否为服务端流式
	ServerStreams bool
}

// Handler trpc默认的handler
type Handler func(ctx context.Context, f FilterFunc) (rspbody interface{}, err error)

// StreamHandlerWapper srv参数为用户实现的流式service的处理入口，stream为srv的参数
type StreamHandlerWapper func(srv interface{}, stream Stream) error

// StreamHandler 流式service处理逻辑，stream参数用来发送和接收流式数据
type StreamHandler func(stream Stream) error

// Stream 服务端流式的接口，用户调用的API入口
type Stream interface {
	// Context 返回当前流的上下文
	Context() context.Context
	// SendMsg 用来发送流式消息
	SendMsg(m interface{}) error
	// RecvMsg 用来接收流式消息
	RecvMsg(m interface{}) error
}

// service Service实现
type service struct {
	ctx            context.Context    // service关闭
	cancel         context.CancelFunc // service关闭
	opts           *Options           // service选项
	handlers       map[string]Handler // rpcname => handler
	streamHandlers map[string]StreamHandler
}

// New 创建一个service 使用全局默认server transport，也可以传参替换
var New = func(opts ...Option) Service {
	s := &service{
		opts: &Options{
			protocol:                 "unknown-protocol",
			ServiceName:              "empty-name",
			CurrentSerializationType: -1,
			CurrentCompressType:      -1,
			Transport:                transport.DefaultServerTransport,
			OverloadCtrl:             overloadctrl.NoopOC{},
		},
		handlers:       make(map[string]Handler),
		streamHandlers: make(map[string]StreamHandler),
	}
	for _, o := range opts {
		o(s.opts)
	}
	if !s.opts.handlerSet { // 没有设置handler 则将该server作为transport的handler
		s.opts.ServeOptions = append(s.opts.ServeOptions, transport.WithHandler(s))
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

// Serve 启动服务
func (s *service) Serve() error {
	pid := os.Getpid()
	// 确保正常监听之后才能启动服务注册
	if err := s.opts.Transport.ListenAndServe(s.ctx, s.opts.ServeOptions...); err != nil {
		log.Errorf("process:%d service:%s ListenAndServe fail:%v", pid, s.opts.ServiceName, err)
		return err
	}

	if s.opts.Registry != nil {
		if err := s.opts.Registry.Register(s.opts.ServiceName, registry.WithAddress(s.opts.Address)); err != nil {
			// 有注册失败，关闭service，并返回给上层错误
			log.Errorf("process:%d, service:%s register fail:%v", pid, s.opts.ServiceName, err)
			return err
		}
	}

	log.Infof("process:%d, %s service:%s launch success, %s:%s, serving ...",
		pid, s.opts.protocol, s.opts.ServiceName, s.opts.network, s.opts.Address)

	report.ServiceStart.Incr()
	<-s.ctx.Done()
	return nil
}

// Handle server transport收到请求包后调用此函数
func (s *service) Handle(ctx context.Context, reqbuf []byte) (rspbuf []byte, err error) {
	// 无法回包，只能丢弃
	if s.opts.Codec == nil {
		log.ErrorContextf(ctx, "server codec empty")
		report.ServerCodecEmpty.Incr()
		return nil, errors.New("server codec empty")
	}

	msg := codec.Message(ctx)
	reqbodybuf, err := s.decode(ctx, msg, reqbuf)
	if err != nil {
		return s.encode(ctx, msg, nil, err)
	}
	// 已经有错误了，在解析完包头拿到 RequestID 后立刻返回客户端。
	if err := msg.ServerRspErr(); err != nil {
		return s.encode(ctx, msg, nil, err)
	}

	var addr string
	if msg.RemoteAddr() != nil {
		addr = msg.RemoteAddr().String()
	}
	token, err := s.opts.OverloadCtrl.Acquire(ctx, addr)
	if err != nil {
		report.TCPServerTransportRequestLimitedByOverloadCtrl.Incr()
		return s.encode(ctx, msg, nil,
			errs.NewFrameError(errs.RetServerOverload, err.Error()))
	}

	rspbody, err := s.handle(ctx, msg, reqbodybuf)
	if err != nil {
		// 不回包
		if err == errs.ErrServerNoResponse {
			token.OnResponse(ctx, nil)
			return nil, err
		}
		defer token.OnResponse(ctx, err)
		// 处理失败 给客户端返回错误码, 忽略rspbody
		report.ServiceHandleFail.Incr()
		return s.encode(ctx, msg, nil, err)
	}
	defer func() { token.OnResponse(ctx, err) }()
	return s.handleResponse(ctx, msg, rspbody)
}

// HandleClose 当连接被关闭时，需要调用该方法。目前仅用于通知流式。
func (s *service) HandleClose(ctx context.Context) error {
	if codec.Message(ctx).ServerRspErr() != nil && s.opts.StreamHandle != nil {
		_, err := s.opts.StreamHandle.StreamHandleFunc(ctx, nil, nil)
		return err
	}
	return nil
}

func (s *service) encode(ctx context.Context, msg codec.Msg, rspbodybuf []byte, e error) (rspbuf []byte, err error) {
	if e != nil {
		msg.WithServerRspErr(e)
	}

	rspbuf, err = s.opts.Codec.Encode(msg, rspbodybuf)
	if err != nil {
		report.ServiceCodecEncodeFail.Incr()
		log.ErrorContextf(ctx, "service:%s encode fail:%v", s.opts.ServiceName, err)
		return nil, err
	}
	return rspbuf, nil
}

// handleStream 处理流式消息的逻辑
func (s *service) handleStream(ctx context.Context, reqbuf []byte, sh StreamHandler,
	opts *Options) (resbody interface{}, err error) {
	if s.opts.StreamHandle != nil {
		return s.opts.StreamHandle.StreamHandleFunc(ctx, sh, reqbuf)
	}
	return nil, errs.NewFrameError(errs.RetServerNoService, "Stream method no Handle")
}

func (s *service) decode(ctx context.Context, msg codec.Msg, reqbuf []byte) ([]byte, error) {
	s.setOpt(msg)
	reqbodybuf, err := s.opts.Codec.Decode(msg, reqbuf)
	if err != nil {
		report.ServiceCodecDecodeFail.Incr()
		return nil, errs.NewFrameError(errs.RetServerDecodeFail, "service codec Decode: "+err.Error())
	}

	// 再赋值一遍，防止decode更改了
	s.setOpt(msg)
	return reqbodybuf, nil
}

func (s *service) setOpt(msg codec.Msg) {
	msg.WithNamespace(s.opts.Namespace)           // server 的命名空间
	msg.WithEnvName(s.opts.EnvName)               // server 的环境
	msg.WithSetName(s.opts.SetName)               // server 的set
	msg.WithCalleeServiceName(s.opts.ServiceName) // 以server角度看，caller是上游，callee是自身
}

func (s *service) handle(ctx context.Context, msg codec.Msg, reqbodybuf []byte) (interface{}, error) {
	// 先判断是否为流式RPC，如果流式的RPC，则交由流式处理器处理
	streamHandler, ok := s.streamHandlers[msg.ServerRPCName()]
	if ok {
		return s.handleStream(ctx, reqbodybuf, streamHandler, s.opts)
	}

	handler, ok := s.handlers[msg.ServerRPCName()]
	if !ok {
		handler, ok = s.handlers["*"] // 支持通配符全匹配转发处理
		if !ok {
			report.ServiceHandleRPCNameInvalid.Incr()
			return nil, errs.NewFrameError(errs.RetServerNoFunc,
				fmt.Sprintf("service handle: rpc name %s invalid, current service:%s",
					msg.ServerRPCName(), msg.CalleeServiceName()))
		}
	}

	timeout := s.opts.Timeout
	if msg.RequestTimeout() > 0 && !s.opts.DisableRequestTimeout { // 可以配置禁用
		if msg.RequestTimeout() < timeout || timeout == 0 { // 取最小值
			timeout = msg.RequestTimeout()
		}
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	rspbody, err := handler(ctx, s.filterFunc(msg, reqbodybuf))
	if err != nil {
		return nil, err
	}
	if msg.CallType() == codec.SendOnly {
		return nil, errs.ErrServerNoResponse
	}
	return rspbody, nil
}

// handleResponse 处理rspbody
func (s *service) handleResponse(ctx context.Context, msg codec.Msg, rspbody interface{}) ([]byte, error) {
	// 默认以协议字段的序列化为准，当有设置option则以option为准
	serializationType := msg.SerializationType()
	compressType := msg.CompressType()
	if s.opts.CurrentSerializationType >= 0 {
		serializationType = s.opts.CurrentSerializationType
	}
	if s.opts.CurrentCompressType >= 0 {
		compressType = s.opts.CurrentCompressType
	}

	// 业务处理成功 才开始打包body
	rspbodybuf, err := codec.Marshal(serializationType, rspbody)
	if err != nil {
		report.ServiceCodecMarshalFail.Incr()
		err = errs.NewFrameError(errs.RetServerEncodeFail, "service codec Marshal: "+err.Error())
		// 处理失败 给客户端返回错误码
		return s.encode(ctx, msg, rspbodybuf, err)
	}

	// 处理成功 才开始压缩body
	rspbodybuf, err = codec.Compress(compressType, rspbodybuf)
	if err != nil {
		report.ServiceCodecCompressFail.Incr()
		err = errs.NewFrameError(errs.RetServerEncodeFail, "service codec Compress: "+err.Error())
		// 处理失败 给客户端返回错误码
		return s.encode(ctx, msg, rspbodybuf, err)
	}

	return s.encode(ctx, msg, rspbodybuf, nil)
}

// filterFunc 生成service拦截器函数 传给server stub，由生成代码来调用该拦截器，以具体业务处理handler前后为拦截入口
func (s *service) filterFunc(msg codec.Msg, reqbodybuf []byte) FilterFunc {
	// 将解压缩，序列化放到该闭包函数内部，允许生成代码里面修改解压缩方式和序列化方式，用于代理层透传
	return func(reqbody interface{}) (filter.Chain, error) {
		// 默认以协议字段的序列化为准，当有设置option则以option为准
		serializationType := msg.SerializationType()
		compressType := msg.CompressType()
		if s.opts.CurrentSerializationType >= 0 {
			serializationType = s.opts.CurrentSerializationType
		}
		if s.opts.CurrentCompressType >= 0 {
			compressType = s.opts.CurrentCompressType
		}

		// 解压body
		reqbodybuf, err := codec.Decompress(compressType, reqbodybuf)
		if err != nil {
			report.ServiceCodecDecompressFail.Incr()
			return nil, errs.NewFrameError(errs.RetServerDecodeFail, "service codec Decompress: "+err.Error())
		}

		// 反序列化body
		err = codec.Unmarshal(serializationType, reqbodybuf, reqbody)
		if err != nil {
			report.ServiceCodecUnmarshalFail.Incr()
			return nil, errs.NewFrameError(errs.RetServerDecodeFail, "service codec Unmarshal: "+err.Error())
		}
		return s.opts.Filters, nil
	}
}

// Register 把service业务实现接口注册到server里面
func (s *service) Register(serviceDesc interface{}, serviceImpl interface{}) error {
	desc, ok := serviceDesc.(*ServiceDesc)
	if !ok {
		return errors.New("serviceDesc is not *ServiceDesc")
	}
	if desc.StreamHandle != nil {
		s.opts.StreamHandle = desc.StreamHandle
		// 流式不能启用idletimout，在连接空闲时候关闭连接。因此重新创建transport，不能使用默认
		s.opts.Transport = transport.NewServerStreamTransport(transport.WithReusePort(true),
			transport.WithIdleTimeout(0))
		err := s.opts.StreamHandle.Init(s.opts)
		if err != nil {
			return err
		}
	}

	if serviceImpl != nil {
		ht := reflect.TypeOf(desc.HandlerType).Elem()
		hi := reflect.TypeOf(serviceImpl)
		if !hi.Implements(ht) {
			return fmt.Errorf("%s not implements interface %s", hi.String(), ht.String())
		}
	}

	var bindings []*restful.Binding
	for _, method := range desc.Methods {
		n := method.Name
		if _, ok := s.handlers[n]; ok {
			return fmt.Errorf("duplicate method name: %s", n)
		}
		h := method.Func
		s.handlers[n] = func(ctx context.Context, f FilterFunc) (rsp interface{}, err error) {
			return h(serviceImpl, ctx, f)
		}
		bindings = append(bindings, method.Bindings...)
	}

	for _, stream := range desc.Streams {
		n := stream.StreamName
		if _, ok := s.streamHandlers[n]; ok {
			return fmt.Errorf("duplicate stream name: %s", n)
		}
		h := stream.Handler
		s.streamHandlers[n] = func(stream Stream) error {
			return h(serviceImpl, stream)
		}
	}

	if len(bindings) > 0 { // 说明有指定 pb option，则创建一个对应的 RESTful Router
		restOpts := s.opts.RESTOptions
		restOpts = append(restOpts, restful.WithServiceName(s.opts.ServiceName)) // service 名
		restOpts = append(restOpts, restful.WithServiceImpl(serviceImpl))        // service 实现
		restOpts = append(restOpts, restful.WithFilterFunc(func() filter.Chain { // service filter chain
			return s.opts.Filters
		}))
		router := restful.NewRouter(restOpts...)
		for _, binding := range bindings {
			if err := router.AddBinding(binding); err != nil {
				return err
			}
		}
		restful.RegisterRouter(s.opts.ServiceName, router)
	}

	return nil
}

// Close service关闭动作，service取消registry注册
func (s *service) Close(ch chan struct{}) error {
	pid := os.Getpid()

	if ch == nil {
		ch = make(chan struct{}, 1)
	}

	log.Infof("process:%d, %s service:%s, closing ...", pid, s.opts.protocol, s.opts.ServiceName)

	if s.opts.Registry != nil {
		err := s.opts.Registry.Deregister(s.opts.ServiceName)
		if err != nil {
			log.Errorf("process:%d, deregister service:%s fail:%v", pid, s.opts.ServiceName, err)
		}
	}
	s.doBeforeClose()

	// service关闭，通知派生的下游ctx全部取消
	s.cancel()
	timeout := time.Millisecond * 300
	if s.opts.Timeout > timeout { // 取最大值
		timeout = s.opts.Timeout
	}
	time.Sleep(timeout)

	log.Infof("process:%d, %s service:%s, closed", pid, s.opts.protocol, s.opts.ServiceName)
	ch <- struct{}{}
	return nil
}

func (s *service) doBeforeClose() {
	closeWaitTime := s.opts.CloseWaitTime
	if closeWaitTime == 0 {
		return
	}

	if closeWaitTime > MaxCloseWaitTime {
		closeWaitTime = MaxCloseWaitTime
	}

	// service关闭时,会先调用反注册来移除本实例,sleep一段时间来避免调用者访问该实例失败
	// 比如k8s更新pod时,调用北极星反注册到调用方从北极星只获取最新pod ip这段时间,需要继续保持服务
	time.Sleep(closeWaitTime)
	log.Infof("wait %v time before really close to make this service smooth for caller", closeWaitTime)
}
