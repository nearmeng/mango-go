package transport

import (
	"context"
	"net"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
	"git.code.oa.com/trpc-go/trpc-go/pool/connpool"
	"git.code.oa.com/trpc-go/trpc-go/pool/multiplexed"
)

// tcpRoundTrip 发送tcp请求 支持 1.send 2. sendAndRcv 3. keepalive 4. multiplex
func (c *clientTransport) tcpRoundTrip(ctx context.Context, reqData []byte,
	opts *RoundTripOptions) ([]byte, error) {
	if opts.Pool == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: connection pool empty")
	}

	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: framer builder empty")
	}

	conn, err := c.dialTCP(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer conn.Close() // tcp连接是独占复用的，Close内部会判断是否应该放回连接池继续复用
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(conn.RemoteAddr())
	msg.WithLocalAddr(conn.LocalAddr())

	if ctx.Err() == context.Canceled {
		return nil, errs.NewFrameError(errs.RetClientCanceled,
			"tcp client transport canceled before Write: "+ctx.Err().Error())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errs.NewFrameError(errs.RetClientTimeout,
			"tcp client transport timeout before Write: "+ctx.Err().Error())
	}

	report.TCPClientTransportSendSize.Set(float64(len(reqData)))
	if err := c.tcpWriteFrame(ctx, conn, reqData); err != nil {
		return nil, err
	}
	return c.tcpReadFrame(conn, opts)
}

// dialTCP 建立 tcp 连接
func (c *clientTransport) dialTCP(ctx context.Context, opts *RoundTripOptions) (net.Conn, error) {
	// 如果ctx已经canceled或者超时，直接返回
	if ctx.Err() == context.Canceled {
		return nil, errs.NewFrameError(errs.RetClientCanceled,
			"client canceled before tcp dial: "+ctx.Err().Error())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errs.NewFrameError(errs.RetClientTimeout,
			"client timeout before tcp dial: "+ctx.Err().Error())
	}
	var timeout time.Duration
	d, ok := ctx.Deadline()
	if ok {
		timeout = time.Until(d)
	}

	var conn net.Conn
	var err error
	// 短连接模式
	if opts.DisableConnectionPool {
		// ctx 超时时间小于设置的连接建立超时，取小的时间进行建立连接。
		if opts.DialTimeout > 0 && opts.DialTimeout < timeout {
			timeout = opts.DialTimeout
		}
		conn, err = connpool.Dial(&connpool.DialOptions{
			Network:       opts.Network,
			Address:       opts.Address,
			LocalAddr:     opts.LocalAddr,
			Timeout:       timeout,
			CACertFile:    opts.CACertFile,
			TLSCertFile:   opts.TLSCertFile,
			TLSKeyFile:    opts.TLSKeyFile,
			TLSServerName: opts.TLSServerName,
		})
		if err != nil {
			return nil, errs.NewFrameError(errs.RetClientConnectFail,
				"tcp client transport dial: "+err.Error())
		}
		if ok {
			conn.SetDeadline(d)
		}
		return conn, nil
	}

	// 连接池模式
	if pool, ok := opts.Pool.(connpool.PoolWithOptions); ok {
		getOpts := connpool.NewGetOptions()
		getOpts.WithContext(ctx)
		getOpts.WithFramerBuilder(opts.FramerBuilder)
		getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
		getOpts.WithLocalAddr(opts.LocalAddr)
		getOpts.WithDialTimeout(opts.DialTimeout)
		conn, err = pool.GetWithOptions(opts.Network, opts.Address, getOpts)
	} else {
		conn, err = opts.Pool.Get(opts.Network, opts.Address, timeout,
			connpool.WithContext(ctx),
			connpool.WithFramerBuilder(opts.FramerBuilder),
			connpool.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName))
	}
	if err != nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport connection pool: "+err.Error())
	}
	if ok {
		conn.SetDeadline(d)
	}
	return conn, nil
}

// tcpWriteReqData tcp 写请求数据
func (c *clientTransport) tcpWriteFrame(ctx context.Context, conn net.Conn, reqData []byte) error {
	// 循环发包
	sentNum := 0
	num := 0
	var err error
	for sentNum < len(reqData) {
		num, err = conn.Write(reqData[sentNum:])
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				return errs.NewFrameError(errs.RetClientTimeout,
					"tcp client transport Write: "+err.Error())
			}
			return errs.NewFrameError(errs.RetClientNetErr,
				"tcp client transport Write: "+err.Error())
		}
		sentNum += num
	}
	return nil
}

// tcpReadFrame tcp 读数据帧
func (c *clientTransport) tcpReadFrame(conn net.Conn, opts *RoundTripOptions) ([]byte, error) {
	// 只发不收
	if opts.ReqType == SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	var fr codec.Framer
	if opts.DisableConnectionPool {
		// 禁用连接池每个链接需要新建 Framer
		fr = opts.FramerBuilder.New(codec.NewReader(conn))
	} else {
		// 连接池中的连接 Framer 和 conn 绑定的
		var ok bool
		fr, ok = conn.(codec.Framer)
		if !ok {
			return nil, errs.NewFrameError(errs.RetClientConnectFail,
				"tcp client transport: framer not implemented")
		}
	}

	rspData, err := fr.ReadFrame()
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp client transport ReadFrame: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp client transport ReadFrame: "+err.Error())
	}
	report.TCPClientTransportReceiveSize.Set(float64(len(rspData)))
	return rspData, nil
}

// tcpMultiplexd 处理多路复用请求
func (c *clientTransport) multiplexed(ctx context.Context, req []byte, opts *RoundTripOptions) ([]byte, error) {
	if opts.Multiplexed == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: connection multiplexed empty")
	}
	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"tcp client transport: framer builder empty")
	}
	getOpts := multiplexed.NewGetOptions()
	getOpts.WithMsg(opts.Msg)
	getOpts.WithFramerBuilder(opts.FramerBuilder)
	getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
	getOpts.WithLocalAddr(opts.LocalAddr)
	conn, err := opts.Multiplexed.Get(ctx, opts.Network, opts.Address, getOpts)
	if err != nil {
		return nil, err
	}
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(conn.RemoteAddr())

	if err := conn.Write(req); err != nil {
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp client multiplexed transport Write: "+err.Error())
	}

	// SendOnly 不需要读回包
	if opts.ReqType == codec.SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	buf, err := conn.Read()
	if err != nil {
		if err == context.Canceled {
			return nil, errs.NewFrameError(errs.RetClientCanceled,
				"tcp client multiplexed transport ReadFrame: "+err.Error())
		}
		if err == context.DeadlineExceeded {
			return nil, errs.NewFrameError(errs.RetClientTimeout,
				"tcp client multiplexed transport ReadFrame: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr,
			"tcp client multiplexed transport ReadFrame: "+err.Error())
	}
	return buf, nil
}
