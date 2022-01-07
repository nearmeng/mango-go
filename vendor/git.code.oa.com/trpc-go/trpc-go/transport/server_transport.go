package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/log"
	reuseport "git.woa.com/trpc-go/go_reuseport"
	"github.com/panjf2000/ants/v2"
)

const (
	// EnvGraceRestart 热重启标识
	EnvGraceRestart = "TRPC_IS_GRACEFUL"

	// EnvGraceFirstFd 第一个listener fd编号
	EnvGraceFirstFd = "TRPC_GRACEFUL_1ST_LISTENFD"

	// EnvGraceRestartFdNum 热重启传递的listener fd数量
	EnvGraceRestartFdNum = "TRPC_GRACEFUL_LISTENFD_NUM"
)

var (
	errUnSupportedListenerType = errors.New("not supported listener type")
	errUnSupportedNetworkType  = errors.New("not supported network type")
	errFileIsNotSocket         = errors.New("file is not a socket")
)

// DefaultServerTransport ServerTransport默认实现
var DefaultServerTransport = NewServerStreamTransport(WithReusePort(true))

// NewServerTransport new出来server transport实现
func NewServerTransport(opt ...ServerTransportOption) ServerTransport {
	// option 默认值
	opts := &ServerTransportOptions{
		RecvUDPPacketBufferSize: 65536,
		SendMsgChannelSize:      100,
		RecvMsgChannelSize:      100,
	}

	for _, o := range opt {
		o(opts)
	}
	addrToConn := make(map[string]*tcpconn)
	return &serverTransport{addrToConn: addrToConn, m: &sync.RWMutex{}, opts: opts}
}

// serverTransport server transport具体实现 包括tcp udp serving
type serverTransport struct {
	addrToConn map[string]*tcpconn
	m          *sync.RWMutex
	opts       *ServerTransportOptions
}

// ListenAndServe 启动监听，如果监听失败则返回错误
func (s *serverTransport) ListenAndServe(ctx context.Context, opts ...ListenServeOption) error {
	lsopts := &ListenServeOptions{}
	for _, opt := range opts {
		opt(lsopts)
	}

	if lsopts.Listener != nil {
		return s.listenAndServeStream(ctx, lsopts)
	}
	// 支持同时监听tcp & udp
	networks := strings.Split(lsopts.Network, ",")
	for _, network := range networks {
		lsopts.Network = network
		switch lsopts.Network {
		case "tcp", "tcp4", "tcp6":
			if err := s.listenAndServeStream(ctx, lsopts); err != nil {
				return err
			}
		case "udp", "udp4", "udp6":
			if err := s.listenAndServePacket(ctx, lsopts); err != nil {
				return err
			}
		default:
			return fmt.Errorf("server transport: not support network type %s", lsopts.Network)
		}
	}
	return nil
}

// ---------------------------------stream server-----------------------------------------//

func (s *serverTransport) getStreamTLSConfig(opts *ListenServeOptions) (*tls.Config, error) {
	var err error
	tlsConf := &tls.Config{}

	if len(opts.CACertFile) != 0 { // 验证客户端证书
		tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
		if opts.CACertFile != "root" {
			ca, err := ioutil.ReadFile(opts.CACertFile)
			if err != nil {
				return nil, fmt.Errorf("Read ca cert file error:%v", err)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(ca) {
				return nil, fmt.Errorf("AppendCertsFromPEM fail")
			}
			tlsConf.ClientCAs = pool
		}
	}
	cert, err := tls.LoadX509KeyPair(opts.TLSCertFile, opts.TLSKeyFile)
	if err != nil {
		return nil, err
	}
	tlsConf.Certificates = []tls.Certificate{cert}
	return tlsConf, nil
}

var (
	listenersMap          = &sync.Map{} // 记录当前服务进程中在用的listeners
	inheritedListenersMap = &sync.Map{} // 记录从父进程继承过来的listeners，同一个key(host:port)对应多个listener fd
	once                  sync.Once     // 控制从父进程传递的fd构建listeners
)

// GetListenersFds 获取服务监听listeners对应的fd
func GetListenersFds() []*ListenFd {
	listenersFds := []*ListenFd{}
	listenersMap.Range(func(key, val interface{}) bool {
		var (
			fd  *ListenFd
			err error
		)

		switch key.(type) {
		case net.Listener:
			fd, err = getListenerFd(key.(net.Listener))
		case net.PacketConn:
			fd, err = getPacketConnFd(key.(net.PacketConn))
		default:
			log.Errorf("listener type passing not supported, type: %T", key)
			err = fmt.Errorf("not supported listener type: %T", key)
		}
		if err != nil {
			log.Errorf("cannot get the listener fd, err: %v", err)
			return true
		}
		listenersFds = append(listenersFds, fd)
		return true
	})
	return listenersFds
}

// SaveListener 保存外部传进来的listener
func SaveListener(listener interface{}) error {
	switch listener.(type) {
	case net.Listener, net.PacketConn:
		listenersMap.Store(listener, true)
	default:
		return fmt.Errorf("not supported listener type: %T", listener)
	}
	return nil
}

// getTCPListener 获取 tcp listener
func (s *serverTransport) getTCPListener(opts *ListenServeOptions) (listener net.Listener, err error) {
	listener = opts.Listener
	var tlsConf *tls.Config

	if listener != nil {
		return listener, nil
	}

	v, _ := os.LookupEnv(EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// find the passed listener
		pln, err := getPassedListener(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}

		listener, ok := pln.(net.Listener)
		if !ok {
			return nil, errors.New("invalid net.Listener")
		}
		return listener, nil
	}

	// 端口重用，内核分发IO ReadReady事件到多核多线程，加速IO效率
	if s.opts.ReusePort {
		listener, err = reuseport.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("tcp reuseport error:%v", err)
		}
	} else {
		listener, err = net.Listen(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}
	}

	// 启用TLS
	if len(opts.TLSCertFile) > 0 && len(opts.TLSKeyFile) > 0 {
		tlsConf, err = s.getStreamTLSConfig(opts)
		if err != nil {
			return nil, err
		}
	}

	if tlsConf != nil {
		listener = tls.NewListener(listener, tlsConf)
	}
	return listener, nil
}

// listenAndServeStream 启动监听，如果监听失败则返回错误
func (s *serverTransport) listenAndServeStream(ctx context.Context, opts *ListenServeOptions) error {
	if opts.FramerBuilder == nil {
		return errors.New("tcp transport FramerBuilder empty")
	}

	listener, err := s.getTCPListener(opts)
	if err != nil {
		return err
	}
	listenersMap.Store(listener, true)

	go s.serveStream(ctx, listener, opts)
	return nil
}

func (s *serverTransport) serveStream(ctx context.Context, ln net.Listener, opts *ListenServeOptions) error {
	return s.serveTCP(ctx, ln, opts)
}

// ---------------------------------packet server-----------------------------------------//

// listenAndServePacket 启动监听，如果监听失败则返回错误
func (s *serverTransport) listenAndServePacket(ctx context.Context, opts *ListenServeOptions) error {
	pool := createUDPRoutinePool(opts.Routines)
	// 端口重用，内核分发IO ReadReady事件到多核多线程，加速IO效率
	if s.opts.ReusePort {
		reuseport.ListenerBacklogMaxSize = 4096
		cores := runtime.NumCPU()
		for i := 0; i < cores; i++ {
			udpconn, err := s.getUDPListener(opts)
			if err != nil {
				return err
			}
			listenersMap.Store(udpconn, true)

			go s.servePacket(ctx, udpconn, pool, opts)
		}
	} else {
		udpconn, err := s.getUDPListener(opts)
		if err != nil {
			return err
		}
		listenersMap.Store(udpconn, true)

		go s.servePacket(ctx, udpconn, pool, opts)
	}
	return nil
}

// getUDPListener 获取 udp listener
func (s *serverTransport) getUDPListener(opts *ListenServeOptions) (udpConn net.PacketConn, err error) {
	v, _ := os.LookupEnv(EnvGraceRestart)
	ok, _ := strconv.ParseBool(v)
	if ok {
		// find the passed listener
		ln, err := getPassedListener(opts.Network, opts.Address)
		if err != nil {
			return nil, err
		}
		listener, ok := ln.(net.PacketConn)
		if !ok {
			return nil, errors.New("invalid net.PacketConn")
		}
		return listener, nil
	}

	if s.opts.ReusePort {
		udpConn, err = reuseport.ListenPacket(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("udp reuseport error:%v", err)
		}
	} else {
		udpConn, err = net.ListenPacket(opts.Network, opts.Address)
		if err != nil {
			return nil, fmt.Errorf("udp listen error:%v", err)
		}
	}

	return udpConn, nil
}

func (s *serverTransport) servePacket(ctx context.Context, rwc net.PacketConn, pool *ants.PoolWithFunc,
	opts *ListenServeOptions) error {
	switch rwc := rwc.(type) {
	case *net.UDPConn:
		return s.serveUDP(ctx, rwc, pool, opts)
	default:
		return errors.New("transport not support PacketConn impl")
	}
}

// ------------------------tcp/udp connection通用结构 统一处理----------------------------//

func (s *serverTransport) newConn(ctx context.Context, opts *ListenServeOptions) *conn {
	idleTimeout := opts.IdleTimeout
	if s.opts.IdleTimeout > 0 {
		idleTimeout = s.opts.IdleTimeout
	}
	return &conn{
		ctx:         ctx,
		handler:     opts.Handler,
		idleTimeout: idleTimeout,
	}
}

// conn 维护连接状态(服务端接收客户端连接请求建立的连接)
type conn struct {
	ctx         context.Context
	cancelCtx   context.CancelFunc
	idleTimeout time.Duration
	lastVisited time.Time
	handler     Handler
}

func (c *conn) handle(ctx context.Context, req []byte) ([]byte, error) {
	return c.handler.Handle(ctx, req)
}

func (c *conn) handleClose(ctx context.Context) error {
	if closeHandler, ok := c.handler.(CloseHandler); ok {
		return closeHandler.HandleClose(ctx)
	}
	return nil
}

var errNotFound = errors.New("listener not found")

// GetPassedListener 根据network+address获取父进程继承过来的listener
func GetPassedListener(network, address string) (interface{}, error) {
	return getPassedListener(network, address)
}

func getPassedListener(network, address string) (interface{}, error) {
	once.Do(inheritListeners)

	key := network + ":" + address
	v, ok := inheritedListenersMap.Load(key)
	if !ok {
		return nil, errNotFound
	}

	listeners := v.([]interface{})
	if len(listeners) == 0 {
		return nil, errNotFound
	}

	ln := listeners[0]
	listeners = listeners[1:]
	if len(listeners) == 0 {
		inheritedListenersMap.Delete(key)
	} else {
		inheritedListenersMap.Store(key, listeners)
	}

	return ln, nil
}

// ListenFd 监听fd描述信息
type ListenFd struct {
	File    *os.File
	Fd      uintptr
	Name    string
	Network string
	Address string
}

// 根据环境变量传过来的起始listenfd以及listenfd数量，来获取listenfd
func inheritListeners() {
	firstListenFd, err := strconv.ParseUint(os.Getenv(EnvGraceFirstFd), 10, 32)
	if err != nil {
		log.Errorf("invalid %s, error: %v", EnvGraceFirstFd, err)
	}

	num, err := strconv.ParseUint(os.Getenv(EnvGraceRestartFdNum), 10, 32)
	if err != nil {
		log.Errorf("invalid %s, error: %v", EnvGraceRestartFdNum, err)
	}

	for fd := firstListenFd; fd <= firstListenFd+num-1; fd++ {
		file := os.NewFile(uintptr(fd), "")
		listener, addr, err := fileListener(file)
		file.Close()
		if err != nil {
			log.Errorf("get file listener error: %v", err)
			continue
		}

		key := addr.Network() + ":" + addr.String()
		v, ok := inheritedListenersMap.LoadOrStore(key, []interface{}{listener})
		if ok {
			listeners := v.([]interface{})
			listeners = append(listeners, listener)
			inheritedListenersMap.Store(key, listeners)
		}
	}
}

func fileListener(file *os.File) (interface{}, net.Addr, error) {
	// 检查file状态
	fin, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	// 检查是否是socket
	if fin.Mode()&os.ModeSocket == 0 {
		return nil, nil, errFileIsNotSocket
	}

	// tcp, tcp4, tcp6
	if listener, err := net.FileListener(file); err == nil {
		return listener, listener.Addr(), nil
	}

	// udp, udp4, udp6
	if packetConn, err := net.FilePacketConn(file); err == nil {
		return packetConn, packetConn.LocalAddr(), nil
	}

	return nil, nil, errUnSupportedNetworkType
}

func getPacketConnFd(conn net.PacketConn) (*ListenFd, error) {
	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		return nil, errors.New("not valid *net.UDPConn")
	}

	file, err := udpConn.File()
	if err != nil {
		return nil, err
	}

	fd := ListenFd{
		File:    file,
		Fd:      file.Fd(),
		Name:    "a udp listener fd",
		Network: udpConn.LocalAddr().Network(),
		Address: udpConn.LocalAddr().String(),
	}
	return &fd, nil
}

func getListenerFd(listener net.Listener) (*ListenFd, error) {
	var (
		file    *os.File
		network string
		address string
		err     error
	)

	switch listener.(type) {
	case *net.TCPListener:
		ln := listener.(*net.TCPListener)
		file, err = ln.File()
		network = ln.Addr().Network()
		address = ln.Addr().String()
	case *net.UnixListener:
		ln := listener.(*net.UnixListener)
		file, err = ln.File()
		network = ln.Addr().Network()
		address = ln.Addr().String()
	default:
		err = errUnSupportedListenerType
	}

	if err != nil {
		return nil, err
	}

	listenFd := ListenFd{
		File:    file,
		Fd:      file.Fd(),
		Name:    "a tcp listener fd",
		Network: network,
		Address: address,
	}
	return &listenFd, nil
}
