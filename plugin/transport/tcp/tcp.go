package tcp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/nearmeng/mango-go/common/uid"
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/plugin/transport"
)

type tcpConn struct {
	connID        uint64
	ctx           context.Context
	lastReadTime  time.Time
	lastWriteTime time.Time
	conn          *net.TCPConn
	localAddr     net.Addr
	remoteAddr    net.Addr
	reader        *bufio.Reader
	writer        *bufio.Writer
	cancleCtx     context.Context
	cancle        context.CancelFunc
}

const (
	_maxBufSize = 512000
)

func NewTcpConn(ctx context.Context, conn *net.TCPConn) *tcpConn {
	cancleCtx, cancle := context.WithCancel(context.Background())

	tcpCtx := &tcpConn{
		ctx:          ctx,
		connID:       uid.GenerateUID(),
		lastReadTime: time.Now(),
		conn:         conn,
		localAddr:    conn.LocalAddr(),
		remoteAddr:   conn.RemoteAddr(),
		reader:       bufio.NewReaderSize(conn, int(_maxBufSize)),
		writer:       bufio.NewWriterSize(conn, int(_maxBufSize)),

		cancleCtx: cancleCtx,
		cancle:    cancle,
	}

	return tcpCtx
}

func (c *tcpConn) GetConnID() uint64 {
	return c.connID
}

func (c *tcpConn) GetLocalAddr() (addr net.Addr) {
	return c.localAddr
}

func (c *tcpConn) GetRemoteAddr() (addr net.Addr) {
	return c.remoteAddr
}

func (c *tcpConn) Send(data []byte) error {
	result, err := transport.GetCodec().Encode(c, data)
	if err != nil {
		log.Error("codec encode failed for %s", err.Error())
		return err
	}

	c.setWriteTimeout()

	n, err := c.writer.Write(result)
	if err != nil {
		return err
	}

	c.writer.Flush()
	log.Info("conn send data size %d to %s", n, c.conn.RemoteAddr().String())

	return nil
}

func (c *tcpConn) setReadTimeout() {
	if _transInst.cfg.IdleTimeout > 0 {
		now := time.Now()
		if now.Sub(c.lastReadTime) > 2*time.Second {
			c.lastReadTime = now
			c.conn.SetReadDeadline(now.Add(time.Duration(_transInst.cfg.IdleTimeout) * time.Second))
		}
	}
}

func (c *tcpConn) setWriteTimeout() {
	if _transInst.cfg.IdleTimeout > 0 {
		now := time.Now()
		if now.Sub(c.lastWriteTime) > 2*time.Second {
			c.lastWriteTime = now
			c.conn.SetWriteDeadline(now.Add(time.Duration(_transInst.cfg.IdleTimeout) * time.Second))
		}
	}
}

func (c *tcpConn) Read(targetBuff []byte) (int, error) {
	targetLen := len(targetBuff)

	if targetLen == 0 {
		return 0, nil
	}

	index := 0
	for index < targetLen {
		n, err := c.reader.Read(targetBuff[index:])
		if err != nil {
			return 0, nil

		}

		index += n
	}
	return index, nil
}

func (c *tcpConn) Recv() {
	defer c.Close(false)

	_transInst.eventHandler.OnConnOpened(c)

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.cancleCtx.Done():
			log.Info("recv logic notify to stop client %s", c.remoteAddr.String())
			return
		default:
		}

		c.setReadTimeout()

		pkg, err := transport.GetCodec().Decode(c)
		if err != nil {
			log.Error("codec decode failed for %s", err.Error())
			return
		}

		_transInst.eventHandler.OnData(c, pkg)
	}

}

func (c *tcpConn) Close(active bool) error {
	var once sync.Once

	once.Do(func() {
		_ = c.writer.Flush()

		_transInst.eventHandler.OnConnClosed(c, active)

		c.cancle()

		_ = c.conn.Close()

	})

	return nil
}

type TcpTransportCfg struct {
	Addr        string `mapstructure:"addr"`
	IdleTimeout uint32 `mapstructure:"idletimeout"`
}

type TcpTransport struct {
	eventHandler transport.EventHandler
	cancel       context.CancelFunc
	cfg          *TcpTransportCfg
}

var (
	_transInst *TcpTransport
)

func NewTcpTransport(cfg *TcpTransportCfg) (*TcpTransport, error) {
	_transInst = &TcpTransport{
		cfg: cfg,
	}
	return _transInst, nil
}

func (t *TcpTransport) SetConfig(cfg *TcpTransportCfg) {
	t.cfg = cfg
}

func (t *TcpTransport) Init(o transport.Options) error {
	t.eventHandler = o.EventHandler

	addr, err := net.ResolveTCPAddr("tcp", t.cfg.Addr)
	if err != nil {
		log.Error("resolve err: %s", err.Error())
		return fmt.Errorf("resolve err:%w", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Error("listen fail for %s", err.Error())
		return fmt.Errorf("listen fail, err:%w", err)
	}

	ctx, cancle := context.WithCancel(context.Background())

	go func() {
		t.serve(ctx, listener)
	}()

	t.cancel = cancle

	return nil
}

func (t *TcpTransport) serve(ctx context.Context, listener *net.TCPListener) {
	log.Info("tcp tranport begin to serve")

	var once sync.Once
	defer once.Do(func() {
		listener.Close()
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := listener.AcceptTCP()
		if err != nil {
			var e net.Error
			if errors.As(err, &e) && e.Temporary() {
				time.Sleep(2 * time.Millisecond)
				continue
			} else {
				return
			}
		}

		conn.SetReadBuffer(int(_maxBufSize))
		conn.SetWriteBuffer(int(_maxBufSize))

		tcpCtx := NewTcpConn(ctx, conn)
		go tcpCtx.Recv()
	}
}

func (t *TcpTransport) Uninit() {
	t.cancel()
}
