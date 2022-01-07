package transport

import (
	"context"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
	"git.code.oa.com/trpc-go/trpc-go/internal/writev"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"github.com/panjf2000/ants/v2"
)

const defaultBufferSize = 128 * 1024

type handleParam struct {
	req   []byte
	c     *tcpconn
	start time.Time
}

func (p *handleParam) reset() {
	p.req = nil
	p.c = nil
	p.start = time.Time{}
}

var handleParamPool = &sync.Pool{
	New: func() interface{} { return new(handleParam) },
}

func createRoutinePool(size int) *ants.PoolWithFunc {
	if size <= 0 {
		size = math.MaxInt32
	}
	pool, err := ants.NewPoolWithFunc(size, func(args interface{}) {
		param, ok := args.(*handleParam)
		if !ok {
			log.Tracef("routine pool args type error, shouldn't happen!")
			return
		}
		report.TCPServerAsyncGoroutineScheduleDelay.Set(float64(time.Since(param.start).Microseconds()))
		if param.c == nil {
			log.Tracef("routine pool tcpconn is nil, shouldn't happen!")
			return
		}
		param.c.handleSync(param.req)
		param.reset()
		handleParamPool.Put(param)
	})
	if err != nil {
		log.Tracef("routine pool create error:%v", err)
		return nil
	}
	return pool
}

func (s *serverTransport) serveTCP(ctx context.Context, ln net.Listener,
	opts *ListenServeOptions) error {
	var once sync.Once
	closeListener := func() {
		ln.Close()
	}
	defer once.Do(closeListener)
	// create a goroutine to watch ctx.Done() channel
	// once Server.Close(), TCP lisnener should be closed immediately
	// and it won't accept any new conn
	go func() {
		<-ctx.Done()
		log.Tracef("recv server close event")
		once.Do(closeListener)
	}()

	// create routine pool when enable ServerAsync
	var pool *ants.PoolWithFunc
	if opts.ServerAsync {
		pool = createRoutinePool(opts.Routines)
	}

	var tempDelay time.Duration
	for {
		rwc, err := ln.Accept()
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
		if tcpConn, ok := rwc.(*net.TCPConn); ok {
			err = tcpConn.SetKeepAlive(true)
			if err != nil {
				log.Tracef("tcp conn set keepalive error:%v", err)
			}
			if s.opts.KeepAlivePeriod > 0 {
				err = tcpConn.SetKeepAlivePeriod(s.opts.KeepAlivePeriod)
				if err != nil {
					log.Tracef("tcp conn set keepalive period error:%v", err)
				}
			}
		}

		tc := &tcpconn{
			conn:        s.newConn(ctx, opts),
			rwc:         rwc,
			fr:          opts.FramerBuilder.New(codec.NewReader(rwc)),
			remoteAddr:  rwc.RemoteAddr(),
			localAddr:   rwc.LocalAddr(),
			serverAsync: opts.ServerAsync,
			writev:      opts.Writev,
			st:          s,
			pool:        pool,
		}
		// 启动writev发包协程
		if tc.writev {
			tc.buffer = writev.NewBuffer()
			tc.closeNotify = make(chan struct{}, 1)
			tc.buffer.Start(tc.rwc, tc.closeNotify)
		}

		// 通过Framer和配置综合判断收包时是否需要拷贝包，防止并发覆盖包内容
		tc.setCopyFrame(opts.CopyFrame)

		key := addrToKey(tc.remoteAddr)
		s.m.Lock()
		s.addrToConn[key] = tc
		s.m.Unlock()

		go tc.serve()
	}
}

// tcpconn 维护tcp连接状态信息(服务端接收客户端请求建立的连接)
type tcpconn struct {
	*conn
	rwc         net.Conn
	fr          codec.Framer
	localAddr   net.Addr
	remoteAddr  net.Addr
	serverAsync bool
	writev      bool
	copyFrame   bool
	closeOnce   sync.Once
	st          *serverTransport
	pool        *ants.PoolWithFunc
	buffer      *writev.Buffer
	closeNotify chan struct{}
}

// close 清理现场，关闭socket连接
func (c *tcpconn) close() {
	c.closeOnce.Do(func() {
		// send error msg to handler
		ctx, msg := codec.WithNewMessage(context.Background())
		msg.WithLocalAddr(c.localAddr)
		msg.WithRemoteAddr(c.remoteAddr)
		e := &errs.Error{
			Type: errs.ErrorTypeFramework,
			Code: errs.RetServerSystemErr,
			Desc: "trpc",
			Msg:  "Server connection closed",
		}
		msg.WithServerRspErr(e)
		// 连接关闭的信息需要交由handler处理
		if err := c.conn.handleClose(ctx); err != nil {
			log.Trace("transport: notify connection close failed", err)
		}
		// 通知关闭Writev发包协程
		if c.writev {
			close(c.closeNotify)
		}

		// remove cache in serverstream transport
		key := addrToKey(c.remoteAddr)
		c.st.m.Lock()
		delete(c.st.addrToConn, key)
		c.st.m.Unlock()

		// finally Close the socket Connection
		c.rwc.Close()
	})
}

// write tcp发包封装函数
func (c *tcpconn) write(p []byte) (int, error) {
	if c.writev {
		return c.buffer.Write(p)
	}
	return c.rwc.Write(p)
}

func (c *tcpconn) serve() {
	defer c.close()
	for {
		// 检查上游是否关闭
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		if c.idleTimeout > 0 {
			now := time.Now()
			if now.Sub(c.lastVisited) > 5*time.Second { // SetReadDeadline性能损耗较严重，每5s才更新一次timeout
				c.lastVisited = now
				err := c.rwc.SetReadDeadline(now.Add(c.idleTimeout))
				if err != nil {
					log.Trace("transport: tcpconn SetReadDeadline fail ", err)
					return
				}
			}
		}

		req, err := c.fr.ReadFrame()
		if err != nil {
			if err == io.EOF {
				report.TCPServerTransportReadEOF.Incr() // 客户端主动断开连接
				return
			}
			if e, ok := err.(net.Error); ok && e.Timeout() { // 客户端超过空闲时间没有发包，服务端主动超时关闭
				report.TCPServerTransportIdleTimeout.Incr()
				return
			}
			report.TCPServerTransportReadFail.Incr()
			log.Trace("transport: tcpconn serve ReadFrame fail ", err)
			return
		}
		report.TCPServerTransportReceiveSize.Set(float64(len(req)))
		// 如果framer不是并发读安全，则需要拷贝一份数据，防止被覆盖
		if c.copyFrame {
			reqCopy := make([]byte, len(req))
			copy(reqCopy, req)
			req = reqCopy
		}

		c.handle(req)
	}
}

func (c *tcpconn) handle(req []byte) {
	if !c.serverAsync || c.pool == nil {
		c.handleSync(req)
		return
	}

	// 使用sync.pool来分配包处理协程入参，减少一次内存分配，提升性能
	args := handleParamPool.Get().(*handleParam)
	args.req = req
	args.c = c
	args.start = time.Now()
	if err := c.pool.Invoke(args); err != nil {
		report.TCPServerTransportJobQueueFullFail.Incr()
		log.Trace("transport: tcpconn serve routine pool put job queue fail ", err)
		c.handleSyncWithErr(req, errs.ErrServerRoutinePoolBusy)
	}
}

func (c *tcpconn) handleSync(req []byte) {
	c.handleSyncWithErr(req, nil)
}

func (c *tcpconn) handleSyncWithErr(req []byte, e error) {
	ctx, msg := codec.WithNewMessage(context.Background())
	defer codec.PutBackMessage(msg)
	msg.WithServerRspErr(e)
	// 记录LocalAddr和RemoteAddr到Context
	msg.WithLocalAddr(c.localAddr)
	msg.WithRemoteAddr(c.remoteAddr)
	rsp, err := c.conn.handle(ctx, req)
	if err != nil {
		if err != errs.ErrServerNoResponse {
			report.TCPServerTransportHandleFail.Incr()
			log.Trace("transport: tcpconn serve handle fail ", err)
			c.close()
			return
		}
		// 服务端主动不回包，直接返回，流式场景下适用
		return
	}
	report.TCPServerTransportSendSize.Set(float64(len(rsp)))
	// 一应一答直接回复
	if _, err = c.write(rsp); err != nil {
		report.TCPServerTransportWriteFail.Incr()
		log.Trace("transport: tcpconn write fail ", err)
		c.close()
		return
	}
}

func (c *tcpconn) setCopyFrame(isCopy bool) {
	// 默认需要拷贝，防止Framer并发读包导致内容被覆盖
	c.copyFrame = true

	// 以下场景可以设置为不拷贝：
	// 场景1： Framer本身能够保证并发读安全
	if codec.IsSafeFramer(c.fr) {
		c.copyFrame = false
		return
	}

	// 场景2： 同步模式并且参数配置不拷贝(非流式)
	if !c.serverAsync && !isCopy {
		c.copyFrame = false
		return
	}
}
