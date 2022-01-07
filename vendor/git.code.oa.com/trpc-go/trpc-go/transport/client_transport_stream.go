package transport

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/pool/connpool"
)

// DefaultClientStreamTransport 默认的客户端流式transport
var DefaultClientStreamTransport = NewClientStreamTransport()
var defaultStreamPool = connpool.NewConnectionPool(connpool.WithForceClose(true))

// NewClientStreamTransport 新建一个流式的transport
func NewClientStreamTransport(opt ...ClientTransportOption) ClientStreamTransport {
	opts := defaultClientTransportOptions()

	// 将传入的func option写到opts字段中
	for _, o := range opt {
		o(opts)
	}
	// 这里是用streamID 去映射连接，在客户端那边去保证每个客户端的streamID递增而且唯一，否则得加上addr的映射
	streamIDToConn := make(map[uint32]*clientConn)
	return &clientStreamTransport{
		clientTransport: clientTransport{opts: opts},
		streamIDToConn:  streamIDToConn,
		m:               &sync.RWMutex{},
	}
}

// clientStreamTransport 兼容原来的client transport，为了不破坏原有的API
type clientStreamTransport struct {
	clientTransport
	streamIDToConn map[uint32]*clientConn
	m              *sync.RWMutex
}

// RoundTrip 兼容原来的transport RoundTrip，直接调用clientTransport的RoundTrip
func (c *clientStreamTransport) RoundTrip(ctx context.Context, req []byte,
	opts ...RoundTripOption) (rsp []byte, err error) {
	return c.clientTransport.RoundTrip(ctx, req, opts...)
}

// response 包含数据部分和error部分
type response struct {
	err  error
	data []byte
}

// clientConn conn的封装，包含接收缓冲区队列
type clientConn struct {
	opts      *RoundTripOptions
	conn      net.Conn
	isClosed  bool
	connDone  chan struct{}
	recvQueue chan *response
}

// getOptions 初始化RoundTripOptions，并做基础检查
func (c *clientStreamTransport) getOptions(ctx context.Context,
	roundTripOpts ...RoundTripOption) (*RoundTripOptions, error) {
	opts := &RoundTripOptions{
		Pool: defaultStreamPool,
	}

	// 将传入的func option写到opts字段中
	for _, o := range roundTripOpts {
		o(opts)
	}

	if opts.Pool == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: connection pool empty")
	}

	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: framer builder empty")
	}
	return opts, nil
}

// Init初始化，先从连接池获取一个连接，初始化相应的缓冲区
func (c *clientStreamTransport) Init(ctx context.Context, roundTripOpts ...RoundTripOption) error {
	opts, err := c.getOptions(ctx, roundTripOpts...)
	if err != nil {
		return err
	}
	// 如果ctx已经canceled或者超时，直接返回
	if ctx.Err() == context.Canceled {
		return errs.NewFrameError(errs.RetClientCanceled,
			"client canceled before tcp dial: "+ctx.Err().Error())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return errs.NewFrameError(errs.RetClientTimeout,
			"client timeout before tcp dial: "+ctx.Err().Error())
	}
	var timeout time.Duration
	d, ok := ctx.Deadline()
	if ok {
		timeout = d.Sub(time.Now())
	}
	// 从连接池中获取连接
	var conn net.Conn
	if ok {
		conn, err = opts.Pool.Get(opts.Network, opts.Address, timeout,
			connpool.WithFramerBuilder(opts.FramerBuilder),
			connpool.WithContext(ctx),
			connpool.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName))
	} else {
		conn, err = opts.Pool.Get(opts.Network, opts.Address, timeout,
			connpool.WithFramerBuilder(opts.FramerBuilder),
			connpool.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName))
	}

	if err != nil {
		return errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport connection pool: "+err.Error())
	}
	msg := codec.Message(ctx)
	streamID := msg.StreamID()
	// 这个队列的长度需要改一下
	var recvQueue chan *response
	if c.opts.TCPRecvQueueSize > 0 {
		recvQueue = make(chan *response, c.opts.TCPRecvQueueSize)
	} else {
		recvQueue = make(chan *response, defaultClientRecvQueueSize)
	}
	cc := &clientConn{opts: opts, conn: conn, recvQueue: recvQueue}
	msg.WithLocalAddr(conn.LocalAddr())
	c.m.Lock()
	c.streamIDToConn[streamID] = cc
	c.m.Unlock()
	connDone := make(chan struct{}, 1)
	cc.connDone = connDone
	go cc.recv() //协程接收请求
	return nil
}

// Send 发送二进制流，选择直接发送
func (c *clientStreamTransport) Send(ctx context.Context, req []byte, roundTripOpts ...RoundTripOption) error {
	msg := codec.Message(ctx)
	streamID := msg.StreamID()
	// StreamID 当前客户端StreamID通过Stream Client生成，全局唯一
	c.m.RLock()
	cc := c.streamIDToConn[streamID]
	c.m.RUnlock()
	if cc == nil {
		return errs.NewFrameError(errs.RetServerSystemErr, "Connection is Closed")
	}
	if _, err := cc.conn.Write(req); err != nil {
		return err
	}
	return nil
}

// getConnect 根据流ID获取clientConn
func (c *clientStreamTransport) getConnect(ctx context.Context, roundTripOpts ...RoundTripOption) (*clientConn, error) {
	msg := codec.Message(ctx)
	streamID := msg.StreamID()
	c.m.RLock()
	cc := c.streamIDToConn[streamID]
	c.m.RUnlock()
	if cc == nil {
		return nil, errs.NewFrameError(errs.RetServerSystemErr, "Stream is not inited yet")
	}
	return cc, nil
}

// Recv 通过流ID获取对应的连接 ，然后通连接的接收队列进行接收
func (c *clientStreamTransport) Recv(ctx context.Context, roundTripOpts ...RoundTripOption) ([]byte, error) {
	cc, err := c.getConnect(ctx, roundTripOpts...)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		if ctx.Err() == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"tcp client transport canceled before Write: "+ctx.Err().Error())
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp client transport timeout before Write: "+ctx.Err().Error())
		}
	case resp := <-cc.recvQueue:
		return resp.data, resp.err
	}
	return nil, nil
}

// readFrame 读出完整的帧并进行判断
func (cc *clientConn) readFrame(fr codec.Framer) error {
	res := &response{}
	rspData, err := fr.ReadFrame()
	if err != nil {
		res.data = nil
		if err == io.EOF {
			//接收到连接关闭，可能是服务端断开连接
			res.err = err
			cc.recvQueue <- res
			return err
		}
		// 网络超时错误
		if e, ok := err.(net.Error); ok && e.Timeout() {
			res.err = errs.NewFrameError(errs.RetClientTimeout,
				"tcp client transport ReadFrame: "+err.Error())
			cc.recvQueue <- res
			return err
		}
		// 读取请求帧错误
		res.err = errs.NewFrameError(errs.RetClientNetErr,
			"tcp client transport ReadFrame: "+err.Error())
		if !cc.isClosed {
			cc.recvQueue <- res
		}
		return err
	}

	// 正常数据返回
	res.data = rspData
	res.err = nil
	cc.recvQueue <- res
	return nil
}

// recv 协程启动，循环接收数据包
func (cc *clientConn) recv() {
	fr, ok := cc.conn.(codec.Framer)
	if !ok {
		return
	}
	for {
		select {
		//判断连接是否已经关闭，关闭则结束流程
		case <-cc.connDone:
			return
		default:
		}
		// 读数据包
		err := cc.readFrame(fr)
		if err != nil {
			return
		}
	}
}

// Close 关闭链接,清理现场
func (c *clientStreamTransport) Close(ctx context.Context) {
	msg := codec.Message(ctx)
	streamID := msg.StreamID()
	c.m.Lock()
	defer c.m.Unlock()
	if clientConn, ok := c.streamIDToConn[streamID]; ok {
		clientConn.recvQueue = nil
		clientConn.isClosed = true
		clientConn.connDone <- struct{}{}
		if clientConn.conn != nil {
			clientConn.conn.Close()
		}
		delete(c.streamIDToConn, streamID)
	}
}
