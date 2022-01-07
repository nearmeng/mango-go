package transport

import (
	"context"
	"errors"
	"math"
	"net"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/internal/packetbuffer"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"github.com/panjf2000/ants/v2"
)

func (s *serverTransport) serveUDP(ctx context.Context, rwc *net.UDPConn, pool *ants.PoolWithFunc,
	opts *ListenServeOptions) error {

	var tempDelay time.Duration
	buf := packetbuffer.New(make([]byte, s.opts.RecvUDPPacketBufferSize))
	fr := opts.FramerBuilder.New(buf)
	copyFrame := !codec.IsSafeFramer(fr)

	for {
		select {
		case <-ctx.Done():
			return errors.New("recv server close event")
		default:
		}

		// 读取数据前清空 buf
		buf.Reset()
		num, raddr, err := rwc.ReadFromUDP(buf.Bytes())
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		// 设置 buf 长度增加 num
		buf.Advance(num)
		req, err := fr.ReadFrame()
		if err != nil {
			report.UDPServerTransportReadFail.Incr()
			log.Trace("transport: udpconn serve ReadFrame fail ", err)
			continue
		}
		report.UDPServerTransportReceiveSize.Set(float64(len(req)))
		if buf.UnRead() > 0 {
			report.UDPServerTransportUnRead.Incr()
			log.Trace("transport: udpconn serve ReadFrame data remaining %d bytes data", buf.UnRead())
			continue
		}

		c := &udpconn{
			conn:       s.newConn(ctx, opts),
			rwc:        rwc,
			remoteAddr: raddr,
		}

		if copyFrame {
			c.req = make([]byte, len(req))
			copy(c.req, req)
		} else {
			c.req = req
		}

		if pool == nil {
			go c.serve()
			continue
		}
		if err := pool.Invoke(c); err != nil {
			report.UDPServerTransportJobQueueFullFail.Incr()
			log.Trace("transport: udpconn serve routine pool put job queue fail ", err)
			go c.serve()
		}
	}
}

// udpconn 维护udp连接状态信息(服务端接收客户端请求建立)
type udpconn struct {
	*conn
	req        []byte
	rwc        *net.UDPConn
	remoteAddr *net.UDPAddr
}

func (c *udpconn) serve() {
	// 生成新的空的通用消息结构数据，并保存到ctx里面
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)

	// 记录LocalAddr和RemoteAddr到Context
	msg.WithLocalAddr(c.rwc.LocalAddr())
	msg.WithRemoteAddr(c.remoteAddr)

	rsp, err := c.handle(ctx, c.req)
	if err != nil {
		if err != errs.ErrServerNoResponse {
			report.UDPServerTransportHandleFail.Incr()
			log.Tracef("udp handle fail:%v", err)
		}
		return
	}

	report.UDPServerTransportSendSize.Set(float64(len(rsp)))
	if _, err := c.rwc.WriteToUDP(rsp, c.remoteAddr); err != nil {
		report.UDPServerTransportWriteFail.Incr()
		log.Tracef("udp write out fail:%v", err)
		return
	}
}

func createUDPRoutinePool(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pool, err := ants.NewPoolWithFunc(size, func(args interface{}) {
		c, ok := args.(*udpconn)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		c.serve()
	})
	if err != nil {
		log.Tracef("routine pool create error:%v", err)
		return nil
	}
	return pool
}
