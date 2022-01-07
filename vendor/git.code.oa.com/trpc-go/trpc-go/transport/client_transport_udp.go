package transport

import (
	"context"
	"net"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/internal/packetbuffer"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/pool/objectpool"
)

const defaultUDPRecvBufSize = 64 * 1024

var udpBufPool = objectpool.NewBytesPool(defaultUDPRecvBufSize)

// udpRoundTrip 发送udp请求
func (c *clientTransport) udpRoundTrip(ctx context.Context, reqData []byte,
	opts *RoundTripOptions) ([]byte, error) {
	if opts.FramerBuilder == nil {
		return nil, errs.NewFrameError(errs.RetClientConnectFail,
			"udp client transport: framer builder empty")
	}

	conn, addr, err := c.dialUDP(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	msg := codec.Message(ctx)
	msg.WithRemoteAddr(addr)
	msg.WithLocalAddr(conn.LocalAddr())

	if ctx.Err() == context.Canceled {
		return nil, errs.NewFrameError(errs.RetClientCanceled,
			"udp client transport canceled before Write: "+ctx.Err().Error())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return nil, errs.NewFrameError(errs.RetClientTimeout,
			"udp client transport timeout before Write: "+ctx.Err().Error())
	}

	report.UDPClientTransportSendSize.Set(float64(len(reqData)))
	if err := c.udpWriteFrame(conn, reqData, addr, opts); err != nil {
		return nil, err
	}
	return c.udpReadFrame(ctx, conn, opts)
}

// udpReadFrame udp 读数据帧
func (c *clientTransport) udpReadFrame(
	ctx context.Context, conn net.PacketConn, opts *RoundTripOptions) ([]byte, error) {
	// 只发不收
	if opts.ReqType == SendOnly {
		return nil, errs.ErrClientNoResponse
	}

	select {
	case <-ctx.Done():
		return nil, errs.NewFrameError(errs.RetClientTimeout, "udp client transport select after Write: "+ctx.Err().Error())
	default:
	}

	recvData := udpBufPool.Get()
	defer udpBufPool.Put(recvData)
	buf := packetbuffer.New(recvData)
	fr := opts.FramerBuilder.New(buf)
	// 收包
	num, _, err := conn.ReadFrom(buf.Bytes())
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return nil, errs.NewFrameError(errs.RetClientTimeout, "udp client transport ReadFrom: "+err.Error())
		}
		return nil, errs.NewFrameError(errs.RetClientNetErr, "udp client transport ReadFrom: "+err.Error())
	}
	if num == 0 {
		return nil, errs.NewFrameError(errs.RetClientNetErr, "udp client transport ReadFrom: num empty")
	}
	// 设置 buf 长度增加 num
	buf.Advance(num)
	req, err := fr.ReadFrame()
	if err != nil {
		report.UDPClientTransportReadFail.Incr()
		log.Trace("transport: udp client transport ReadFrame fail ", err)
	}
	if buf.UnRead() > 0 {
		report.UDPClientTransportUnRead.Incr()
		log.Trace("transport: udp client ReadFrame data remaining %d bytes data", buf.UnRead())
	}
	report.UDPClientTransportReceiveSize.Set(float64(len(req)))
	// 每次请求都用 framer 所以没必要去拷贝内存
	return req, nil
}

// udpWriteReqData udp 写请求数据
func (c *clientTransport) udpWriteFrame(conn net.PacketConn,
	reqData []byte, addr *net.UDPAddr, opts *RoundTripOptions) error {
	// 发包
	var num int
	var err error
	if opts.ConnectionMode == Connected {
		udpconn := conn.(*net.UDPConn)
		num, err = udpconn.Write(reqData)
	} else {
		num, err = conn.WriteTo(reqData, addr)
	}
	if err != nil {
		if e, ok := err.(net.Error); ok && e.Timeout() {
			return errs.NewFrameError(errs.RetClientTimeout, "udp client transport WriteTo: "+err.Error())
		}
		return errs.NewFrameError(errs.RetClientNetErr, "udp client transport WriteTo: "+err.Error())
	}
	if num != len(reqData) {
		return errs.NewFrameError(errs.RetClientNetErr, "udp client transport WriteTo: num mismatch")
	}
	return nil
}

// dialUDP 建立 udp
func (c *clientTransport) dialUDP(ctx context.Context, opts *RoundTripOptions) (net.PacketConn, *net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr(opts.Network, opts.Address)
	if err != nil {
		return nil, nil, errs.NewFrameError(errs.RetClientNetErr,
			"udp client transport ResolveUDPAddr: "+err.Error())
	}

	var conn net.PacketConn
	if opts.ConnectionMode == Connected {
		var localAddr net.Addr
		if opts.LocalAddr != "" {
			localAddr, err = net.ResolveUDPAddr(opts.Network, opts.LocalAddr)
			if err != nil {
				return nil, nil, errs.NewFrameError(errs.RetClientNetErr,
					"udp client transport LocalAddr ResolveUDPAddr: "+err.Error())
			}
		}
		dialer := net.Dialer{
			LocalAddr: localAddr,
		}
		var udpConn net.Conn
		udpConn, err = dialer.Dial(opts.Network, opts.Address)
		var ok bool
		conn, ok = udpConn.(net.PacketConn)
		if !ok {
			return nil, nil, errs.NewFrameError(errs.RetClientConnectFail,
				"udp conn not implement net.PacketConn")
		}
	} else {
		const defaultLocalAddr = ":"
		localAddr := defaultLocalAddr
		if opts.LocalAddr != "" {
			localAddr = opts.LocalAddr
		}
		conn, err = net.ListenPacket(opts.Network, localAddr)
	}
	if err != nil {
		return nil, nil, errs.NewFrameError(errs.RetClientNetErr, "udp client transport Dial: "+err.Error())
	}
	d, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(d)
	}
	return conn, addr, nil
}
