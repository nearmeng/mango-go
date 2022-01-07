package client

import (
	"context"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
)

// Stream 流式的接口，对应一应一答的Client
type Stream interface {
	// Send 发送流式消息
	Send(ctx context.Context, m interface{}) error
	// Recv 接收流式消息
	Recv(ctx context.Context) ([]byte, error)
	// Init 初始化流
	Init(ctx context.Context, opt ...Option) (*Options, error)
	// Close 关闭流式
	Close(ctx context.Context) error
}

// stream Stream interface 的实现
type stream struct {
	opts *Options
	client
}

// SendControl 流控发送端的相关实现
type SendControl interface {
	GetWindow(uint32) error
	UpdateWindow(uint32)
}

// RecvControl  流控接收端统计
type RecvControl interface {
	OnRecv(n uint32) error
}

// Send 将消息序列化后，通过stream transport发送给服务端
// Recv 和 Send 在不同协程是并发安全的，多个协程一起Send不是并发安全的
func (s *stream) Send(ctx context.Context, m interface{}) error {
	msg := codec.Message(ctx)
	reqbodybuf, err := serializeAndCompress(msg, m, s.opts)
	// 发起后端网络请求
	if err != nil {
		s.opts.StreamTransport.Close(ctx)
		return err
	}
	// 接口数据类型不是nil，说明是Data类型的帧，需要调用流控进行控制
	if m != nil && s.opts.SControl != nil {
		if err := s.opts.SControl.GetWindow(uint32(len(reqbodybuf))); err != nil {
			return err
		}
	}
	// 打包整个请求buffer
	reqbuf, err := s.opts.Codec.Encode(msg, reqbodybuf)
	if err != nil {
		return errs.NewFrameError(errs.RetClientEncodeFail, "client codec Encode: "+err.Error())
	}

	if err := s.opts.StreamTransport.Send(ctx, reqbuf); err != nil {
		s.opts.StreamTransport.Close(ctx)
		return err
	}
	return nil
}

// Recv 接收流式消息，进行解码，解压缩，流式消息是用户传入的，Serialization交由上层做
// Recv 和 Send 在不同协程是并发安全的，多个协程一起Send不是并发安全的
func (s *stream) Recv(ctx context.Context) ([]byte, error) {
	rspbuf, err := s.opts.StreamTransport.Recv(ctx)
	if err != nil {
		s.opts.StreamTransport.Close(ctx)
		return nil, err
	}
	msg := codec.Message(ctx)
	rspbodybuf, err := s.opts.Codec.Decode(msg, rspbuf)
	if err != nil {
		s.opts.StreamTransport.Close(ctx)
		return nil, errs.NewFrameError(errs.RetClientDecodeFail, "client codec Decode: "+err.Error())
	}
	if len(rspbodybuf) > 0 {
		// Data帧类型，统计下流控信息
		if s.opts.RControl != nil {
			if err := s.opts.RControl.OnRecv(uint32(len(rspbodybuf))); err != nil {
				return nil, err
			}
		}

		compressType := msg.CompressType()
		if s.opts.CurrentCompressType >= 0 {
			compressType = s.opts.CurrentCompressType
		}
		// 解压缩
		if compressType > 0 {
			rspbodybuf, err = codec.Decompress(compressType, rspbodybuf)
			if err != nil {
				s.opts.StreamTransport.Close(ctx)
				return nil,
					errs.NewFrameError(errs.RetClientDecodeFail, "client codec Decompress: "+err.Error())
			}
		}
	}
	return rspbodybuf, nil
}

// Close 关闭流
func (s *stream) Close(ctx context.Context) error {
	// 发送Close消息
	return s.Send(ctx, nil)
}

// Init 目标选择合适的地址，获取连接，发送init消息
func (s *stream) Init(ctx context.Context, opt ...Option) (*Options, error) {
	// 取出当前请求链路的通用消息结构数据, 每个client后端调用都是新的msg，由client stub创建生成
	msg := codec.Message(ctx)

	// 读取配置参数，设置用户输入参数
	opts, err := s.getOptions(msg, opt...)
	if err != nil {
		return nil, err
	}
	// 根据寻址选择器寻址到后端节点node
	if _, err = s.selectNode(msg, opts); err != nil {
		report.SelectNodeFail.Incr()
		return nil, err
	}
	if opts.Codec == nil {
		report.ClientCodecEmpty.Incr()
		return nil, errs.NewFrameError(errs.RetClientEncodeFail, "client: codec empty")
	}
	// 根据获取的Opts信息更新Msg
	s.updateMsg(msg, opts)
	s.opts = opts
	err = s.opts.StreamTransport.Init(ctx, opts.CallOptions...)
	return opts, err
}

// DefaultStream 默认的客户端流式 Client
var DefaultStream = NewStream()

// NewStream 返回一个流式 Client
var NewStream = func() Stream {
	return &stream{}
}
