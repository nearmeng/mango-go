package multiplexed

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/internal/packetbuffer"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/pool/connpool"
)

// DefaultMultiplexedPool 默认的多路复用实现
var DefaultMultiplexedPool = New()

const (
	defaultBufferSize    = 128 * 1024
	defaultConnectNumber = 2
	defaultQueueSize     = 1024
	defaultDialTimeout   = 200 * time.Millisecond
	maxBufferSize        = 65535
	initialBackoff       = 5 * time.Millisecond
	maxBackoff           = 50 * time.Millisecond
	maxReconnectCount    = 10
)

var (
	// ErrFrameBuilderNil  framer builder 没有设置
	ErrFrameBuilderNil = errors.New("framer builder is nil")
	// ErrDecoderNil 没有实现 Decoder
	ErrDecoderNil = errors.New("framer do not implement Decoder interface")
	// ErrQueueFull 队列满了
	ErrQueueFull = errors.New("connection queue is full")
	// ErrChanClose 连接关闭
	ErrChanClose = errors.New("unexpected recv chan close")
	// ErrAssertFail 断言类型错误
	ErrAssertFail = errors.New("assert connection slice fail")
	// ErrWriteNotFinished 写操作没完成
	ErrWriteNotFinished = errors.New("write not finished")
	// ErrNetworkNotSupport 不支持网络类型
	ErrNetworkNotSupport = errors.New("network not support")
)

// recvReader 连接复用时用于记录收到的多个回包
type recvReader struct {
	ctx  context.Context
	recv chan []byte
}

// New new multiplexed instance
func New(opt ...PoolOption) *Multiplexed {
	opts := &PoolOptions{
		connectNumber: defaultConnectNumber,
		queueSize:     defaultQueueSize,
		dialTimeout:   time.Second,
	}
	for _, o := range opt {
		o(opts)
	}
	m := &Multiplexed{
		connections: new(sync.Map),
		opts:        opts,
	}
	return m
}

// Multiplexed 多路复用
type Multiplexed struct {
	connections *sync.Map
	opts        *PoolOptions
	mu          sync.RWMutex
}

// Get 获取多路复用对应的虚拟连接
func (p *Multiplexed) Get(ctx context.Context, network string,
	address string, opts GetOptions) (*VirtualConnection, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	requestID := opts.Msg.RequestID()
	if opts.FramerBuilder == nil {
		return nil, ErrFrameBuilderNil
	}

	var isStream bool
	switch network {
	case "tcp", "tcp4", "tcp6":
		isStream = true
	case "udp", "udp4", "udp6":
	default:
		return nil, ErrNetworkNotSupport
	}

	c, err := p.get(network, address, &opts, isStream)
	if err != nil {
		return nil, err
	}
	return c.new(ctx, requestID, opts.Msg)
}

func (p *Multiplexed) get(network, address string,
	opts *GetOptions, isStream bool) (*Connection, error) {
	key := getNodeKey(network, address)
	// 判断是否已经存在连接，如果存在就返回，避免使用锁
	if v, ok := p.connections.Load(key); ok {
		if connections, ok := v.(*Connections); ok {
			return connections.getConnection(), nil
		}
		return nil, ErrAssertFail
	}
	// 当address对应的连接还没创建，需要进行连接创建
	p.mu.Lock()
	// 这里再判断一次，如果前面的请求已经创建了连接，这里直接取就行了
	if v, ok := p.connections.Load(key); ok {
		if connections, ok := v.(*Connections); ok {
			c := connections.getConnection()
			p.mu.Unlock()
			return c, nil
		}
		p.mu.Unlock()
		return nil, ErrAssertFail
	}
	// 连接不存在，这里创建连接
	connections := &Connections{multiplexed: p}
	cs := make([]*Connection, p.opts.connectNumber)
	for i := 0; i < p.opts.connectNumber; i++ {
		c := &Connection{
			address:     address,
			network:     network,
			cs:          connections,
			connections: make(map[uint32]*VirtualConnection),
			done:        make(chan struct{}),
			dropFull:    p.opts.dropFull,
			isStream:    isStream,
		}
		c.buffer = newBuffer(c.done, p.opts.queueSize)
		cs[i] = c
	}
	connections.cs = cs
	p.connections.Store(key, connections)
	p.mu.Unlock()
	p.startConnection(connections, network, address, opts)
	return connections.getConnection(), nil
}

func dialStream(network, address string,
	timeout time.Duration, opts *GetOptions) (net.Conn, *connpool.DialOptions, error) {
	dialOpts := &connpool.DialOptions{
		Network:       network,
		Address:       address,
		Timeout:       timeout,
		CACertFile:    opts.CACertFile,
		TLSCertFile:   opts.TLSCertFile,
		TLSKeyFile:    opts.TLSKeyFile,
		TLSServerName: opts.TLSServerName,
		LocalAddr:     opts.LocalAddr,
	}
	conn, err := tryConnect(dialOpts)
	return conn, dialOpts, err
}

func dialPacket(network, address string,
	opts *GetOptions) (net.PacketConn, *net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, nil, err
	}
	const defaultLocalAddr = ":"
	localAddr := defaultLocalAddr
	if opts.LocalAddr != "" {
		localAddr = opts.LocalAddr
	}
	conn, err := net.ListenPacket(network, localAddr)
	if err != nil {
		return nil, nil, err
	}
	return conn, addr, nil
}

func (cs *Connections) getConnection() *Connection {
	numberConnection := len(cs.cs)
	current := atomic.AddInt64(&cs.number, 1) % int64(numberConnection)
	connection := cs.cs[current]
	return connection
}

type dialFunc func(context.Context) (net.Conn, *connpool.DialOptions, error)

// startConnection 开始真正执行连接逻辑
func (p *Multiplexed) startConnection(cs *Connections,
	network, address string, opts *GetOptions) {
	for i := 0; i < p.opts.connectNumber; i++ {
		go func(c *Connection) {
			c.builder = opts.FramerBuilder
			framer, err := c.newFramer(network, address, p.opts.dialTimeout, opts)
			if err != nil {
				c.close(err, true)
				return
			}
			c.copyFrame = !codec.IsSafeFramer(framer)
			decoder, ok := framer.(codec.Decoder)
			if !ok {
				c.close(ErrDecoderNil, false)
				return
			}
			c.decoder = decoder
			go c.reader()
			go c.writer()
		}(cs.cs[i])
	}
}

func (c *Connection) newFramer(network,
	address string, timeout time.Duration, opts *GetOptions) (codec.Framer, error) {
	var reader io.Reader
	if c.isStream {
		conn, dialOpts, err := dialStream(network, address, timeout, opts)
		c.dialOpts = dialOpts
		if err != nil {
			return nil, err
		}
		c.conn = conn
		reader = codec.NewReaderSize(conn, defaultBufferSize)
	} else {
		conn, addr, err := dialPacket(network, address, opts)
		if err != nil {
			return nil, err
		}
		c.addr = addr
		c.packetConn = conn
		c.packetReader = packetbuffer.New(make([]byte, maxBufferSize))
		reader = c.packetReader
	}
	return c.builder.New(reader), nil
}

func (c *Connection) decodePacket() (codec.TransportResponseFrame, error) {
	c.packetReader.Reset()
	n, _, err := c.packetConn.ReadFrom(c.packetReader.Bytes())
	if err != nil {
		return nil, err
	}
	c.packetReader.Advance(n)

	// try decode packet
	response, err := c.decoder.Decode()
	if err != nil {
		return nil, err
	}

	// if data remaining, it's invalid packet, skip it.
	if c.packetReader.UnRead() > 0 {
		return nil, errors.New("remaining data in buffer")
	}

	return response, nil
}

func (c *Connection) decodeStream() (codec.TransportResponseFrame, error) {
	return c.decoder.Decode()
}

func (c *Connection) decode() (codec.TransportResponseFrame, error) {
	if c.isStream {
		return c.decodeStream()
	}
	return c.decodePacket()
}

func (c *Connection) reader() {
	var lastErr error
	for {
		select {
		case <-c.done:
			return
		default:
		}
		response, err := c.decode()
		if err != nil {
			// tcp 如果解包出错，可能会导致后面的所有解析都出现问题
			// 所以需关闭重连。
			if c.isStream {
				lastErr = err
				break
			}
			// udp 按照单个包处理，收到一个非法包不影响后续的包处理
			// 逻辑，可以继续收包。
			log.Tracef("decode packet err: %s", err)
			continue
		}
		requestID := response.GetRequestID()
		c.mu.RLock()
		vc, ok := c.connections[requestID]
		c.mu.RUnlock()
		if !ok {
			continue
		}
		vc.recv(response)
	}
	c.close(lastErr, true)
}

func (c *Connection) writer() {
	var lastErr error
L:
	for {
		select {
		case <-c.done:
			break L
		case it := <-c.buffer.buffer:
			if err := c.writeAll(it); err != nil {
				// tcp 写数据失败，将会导致对端关闭连接，
				// 所以关闭关闭重连。
				if c.isStream {
					lastErr = err
					break L
				}
				// udp 发包失败，可以继续收包。
				continue
			}
		}
	}
	c.close(lastErr, true)
}

// Connection underlying tcp connection
type Connection struct {
	cs          *Connections
	err         error
	address     string
	network     string
	mu          sync.RWMutex
	connections map[uint32]*VirtualConnection
	decoder     codec.Decoder
	copyFrame   bool
	done        chan struct{} // closed when underlying connection closed.
	buffer      *writerBuffer
	builder     codec.FramerBuilder
	dropFull    bool

	// udp only
	packetReader *packetbuffer.PacketBuffer
	addr         *net.UDPAddr
	packetConn   net.PacketConn // underlying udp connection

	// tcp only
	conn     net.Conn // underlying tcp connection
	dialOpts *connpool.DialOptions
	isStream bool
}

// Connections Connection的集合
type Connections struct {
	cs          []*Connection
	number      int64
	multiplexed *Multiplexed
}

// writerBuffer 写 buffer，连接复用时用于记录待发送的请求
type writerBuffer struct {
	buffer chan []byte
	done   <-chan struct{}
}

func newBuffer(done chan struct{}, size int) *writerBuffer {
	return &writerBuffer{buffer: make(chan []byte, size), done: done}
}

func (c *Connection) new(ctx context.Context, requestID uint32, msg codec.Msg) (*VirtualConnection, error) {
	vc := &VirtualConnection{
		msg:       msg,
		requestID: requestID,
		conn:      c,
		reader: &recvReader{
			ctx:  ctx,
			recv: make(chan []byte, 1),
		},
	}
	c.mu.Lock()
	// 考虑request id溢出问题或者上层 request id 重复，需要先读一下判断request id
	// 是否已经存在，存在的话需要给原来的virtual connection 返回error
	if prevConn, ok := c.connections[requestID]; ok {
		close(prevConn.reader.recv)
	}
	c.connections[requestID] = vc
	c.mu.Unlock()
	return vc, nil
}

func (c *Connection) send(b []byte) error {
	//如果设置dropfull，队列满，则丢弃
	if c.dropFull {
		select {
		case c.buffer.buffer <- b:
			return nil
		default:
			return ErrQueueFull
		}
	}
	c.buffer.buffer <- b
	return nil
}

func (c *Connection) writeAll(b []byte) error {
	if c.isStream {
		return c.writeStream(b)
	}
	return c.writePacket(b)
}

func (c *Connection) writePacket(b []byte) error {
	num, err := c.packetConn.WriteTo(b, c.addr)
	if err != nil {
		return err
	}
	if num != len(b) {
		return ErrWriteNotFinished
	}
	return nil
}

func (c *Connection) writeStream(b []byte) error {
	var sentNum, num int
	var err error
	for sentNum < len(b) {
		num, err = c.conn.Write(b[sentNum:])
		if err != nil {
			return err
		}
		sentNum += num
	}
	return nil
}

func (c *Connection) close(lastErr error, reconnect bool) {
	if c.isStream {
		c.closeStream(lastErr, reconnect)
		return
	}
	c.closePacket(lastErr)
}

func (c *Connection) closePacket(lastErr error) {
	c.cs.multiplexed.connections.Delete(getNodeKey(c.network, c.address))
	c.err = lastErr
	close(c.done)
}

func (c *Connection) closeStream(lastErr error, reconnect bool) {
	if lastErr == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// c.err not nil means already closed, return directly.
	if c.err != nil {
		return
	}
	c.err = lastErr

	// when close the `c.done` channel, all Read operation will return error,
	// so we should clean all existing connections, avoiding memory leak.
	c.connections = make(map[uint32]*VirtualConnection)
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
	}
	if reconnect {
		c.reconnect()
	}
}

func tryConnect(opts *connpool.DialOptions) (net.Conn, error) {
	conn, err := connpool.Dial(opts)
	if err != nil {
		return nil, err
	}
	if c, ok := conn.(*net.TCPConn); ok {
		c.SetKeepAlive(true)
	}
	return conn, nil
}

func (c *Connection) reconnect() {
	multiplier := 0
	for {
		conn, err := tryConnect(c.dialOpts)
		if err != nil {
			report.MultiplexedTCPReconnectErr.Incr()
			log.Tracef("reconnect fail: %+v", err)
			multiplier += 1
			if multiplier > maxReconnectCount {
				// 如果重试次数大于最大重试次数，则移除对应的连接，
				// 新请求会触发重建连接。
				c.destroy()
				break
			}
			currentBackoff := time.Duration(multiplier) * initialBackoff
			if currentBackoff > maxBackoff {
				currentBackoff = maxBackoff
			}
			time.Sleep(currentBackoff)
			continue
		}
		framer := c.builder.New(codec.NewReaderSize(conn, defaultBufferSize))
		// 初始化连接逻辑保证了 framer 实现 codec.Decoder 接口
		// 重连这里直接忽略类型断言结果。
		decoder, _ := framer.(codec.Decoder)
		c.decoder = decoder
		c.conn = conn
		c.done = make(chan struct{})
		c.buffer.done = c.done
		// connection重连成功，解除err标志
		c.err = nil
		go c.reader()
		go c.writer()
		return
	}
}

// destroy 移除当前连接
func (c *Connection) destroy() {
	key := getNodeKey(c.network, c.address)
	c.cs.multiplexed.connections.Delete(key)
}

func (c *Connection) remove(id uint32) {
	c.mu.Lock()
	delete(c.connections, id)
	c.mu.Unlock()
}

// VirtualConnection 多路复用虚拟连接
type VirtualConnection struct {
	requestID uint32
	conn      *Connection
	msg       codec.Msg
	reader    *recvReader
}

// RemoteAddr 获取连接的对端地址
func (vc *VirtualConnection) RemoteAddr() net.Addr {
	if !vc.conn.isStream {
		return vc.conn.addr
	}
	if vc.conn == nil || vc.conn.conn == nil {
		return nil
	}
	return vc.conn.conn.RemoteAddr()
}

// recv 接收返回的数据
func (vc *VirtualConnection) recv(rsp codec.TransportResponseFrame) {
	vc.conn.decoder.UpdateMsg(rsp, vc.msg)
	if vc.conn.copyFrame {
		copyData := make([]byte, len(rsp.GetResponseBuf()))
		copy(copyData, rsp.GetResponseBuf())
		vc.reader.recv <- copyData
	} else {
		vc.reader.recv <- rsp.GetResponseBuf()
	}
	vc.conn.remove(rsp.GetRequestID())
}

// Write 写入请求包
func (vc *VirtualConnection) Write(b []byte) error {
	if err := vc.conn.send(b); err != nil {
		// clean the virtual connection when send fail.
		vc.conn.remove(vc.requestID)
		return err
	}
	return nil
}

// Read 获取回包
func (vc *VirtualConnection) Read() ([]byte, error) {
	select {
	case <-vc.reader.ctx.Done():
		// clean the virtual connection when context timeout.
		vc.conn.remove(vc.requestID)
		return nil, vc.reader.ctx.Err()
	case v, ok := <-vc.reader.recv:
		if ok {
			return v, nil
		}
		if vc.conn.err != nil {
			return nil, vc.conn.err
		}
		return nil, ErrChanClose
	case <-vc.conn.done:
		// all existing connections has been destroyed, no need
		// to remove virtual connection again.
		return nil, vc.conn.err
	}
}

func getNodeKey(network, address string) string {
	var key strings.Builder
	key.Grow(len(network) + len(address) + 1)
	key.WriteString(network)
	key.WriteString("_")
	key.WriteString(address)
	return key.String()
}
