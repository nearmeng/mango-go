package connpool

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/internal/report"
)

const (
	defaultDialTimeout   = 200 * time.Millisecond
	defaultIdleTimeout   = 50 * time.Second
	defaultMaxIdle       = 65536
	defaultCheckInterval = 3 * time.Second
)

var globalBuffer []byte = make([]byte, 1)

// DefaultConnectionPool 默认的连接池，可替换
var DefaultConnectionPool = NewConnectionPool()

// 连接池错误信息
var (
	ErrPoolLimit  = errors.New("connection pool limit")  // ErrPoolLimit 连接数量超过限制错误
	ErrPoolClosed = errors.New("connection pool closed") // ErrPoolClosed 连接池关闭错误
	ErrConnClosed = errors.New("conn closed")            // ErrConnClosed 连接关闭
	ErrNoDeadline = errors.New("dial no deadline")       // ErrNoDeadline 没有设置 deadline
)

// healthChecker 空闲连接健康检查函数
// 函数支持快速检查和全面检查，快速检查在获取空闲连接时调用，只检查连接状态是否异常
// 函数返回 true 表示连接正常可用
type healthChecker func(pc *PoolConn, isFast bool) bool

// NewConnectionPool 创建一个连接池
func NewConnectionPool(opt ...Option) Pool {
	// 默认值, 暂定，需要调试确定具体数值
	opts := &Options{
		MaxIdle:     defaultMaxIdle,
		IdleTimeout: defaultIdleTimeout,
		DialTimeout: defaultDialTimeout,
	}
	for _, o := range opt {
		o(opts)
	}
	return &pool{
		opts:            opts,
		connectionPools: new(sync.Map),
	}
}

// pool 连接池厂，维护所有address对应的连接池，及连接池选项信息
type pool struct {
	opts            *Options
	connectionPools *sync.Map
}

// Get 连接池中获取连接
// Deprecated: please use GetWithOptions () instead.
func (p *pool) Get(network string, address string, _ time.Duration, opt ...GetOption) (net.Conn, error) {
	opts := GetOptions{}
	for _, o := range opt {
		o(&opts)
	}
	return p.GetWithOptions(network, address, opts)
}

type dialFunc = func(ctx context.Context) (net.Conn, error)

func (p *pool) getDialFunc(network string, address string, opts GetOptions) dialFunc {
	dialOpts := &DialOptions{
		Network:       network,
		Address:       address,
		LocalAddr:     opts.LocalAddr,
		CACertFile:    opts.CACertFile,
		TLSCertFile:   opts.TLSCertFile,
		TLSKeyFile:    opts.TLSKeyFile,
		TLSServerName: opts.TLSServerName,
	}

	return func(ctx context.Context) (net.Conn, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		d, ok := ctx.Deadline()
		if !ok {
			return nil, ErrNoDeadline
		}
		dialOpts.Timeout = time.Until(d)
		return Dial(dialOpts)
	}
}

// GetWithOptions  连接池中获取连接
func (p *pool) GetWithOptions(network string, address string, opts GetOptions) (net.Conn, error) {
	ctx, cancel := opts.getDialCtx(p.opts.DialTimeout)
	if cancel != nil {
		defer cancel()
	}
	if v, ok := p.connectionPools.Load(address); ok {
		return v.(*ConnectionPool).Get(ctx)
	}
	newPool := &ConnectionPool{
		Dial:            p.getDialFunc(network, address, opts),
		MinIdle:         p.opts.MinIdle,
		MaxIdle:         p.opts.MaxIdle,
		MaxActive:       p.opts.MaxActive,
		Wait:            p.opts.Wait,
		MaxConnLifetime: p.opts.MaxConnLifetime,
		IdleTimeout:     p.opts.IdleTimeout,
		framerBuilder:   opts.FramerBuilder,
		forceClosed:     p.opts.ForceClose,
	}
	newPool.checker = newPool.defaultChecker

	// 规避初始化连接池map并发写的问题
	v, ok := p.connectionPools.LoadOrStore(address, newPool)
	if !ok {
		newPool.RegisterChecker(defaultCheckInterval, newPool.defaultChecker)
		newPool.initialConnections(newPool.MinIdle)
		return newPool.Get(ctx)
	}
	return v.(*ConnectionPool).Get(ctx)
}

// ConnectionPool 连接池
type ConnectionPool struct {
	Dial            func(context.Context) (net.Conn, error) // 初始化连接
	MinIdle         int                                     // 最小空闲连接数量
	MaxIdle         int                                     // 最大闲置连接数量，0 代表不做限制
	MaxActive       int                                     // 最大活跃连接数量，0 代表不做限制
	IdleTimeout     time.Duration                           // 空闲连接超时时间
	Wait            bool                                    // 活跃连接达到最大数量时，是否等待
	MaxConnLifetime time.Duration                           // 连接的最大生命周期
	mu              sync.Mutex                              // 控制并发的锁
	checker         healthChecker                           // 空闲连接健康检查函数
	closed          bool                                    // 连接池是否已经关闭
	active          int                                     // 目前活跃连接数量
	ch              chan struct{}                           // 当 Wait 为 true 的时候，用来限制连接数量
	initChOnce      sync.Once                               // 表明 ch 是否已经初始化
	idle            connList                                // 空闲连接链表
	framerBuilder   codec.FramerBuilder
	forceClosed     bool // 强制关闭连接，适用于流式场景
}

func (p *ConnectionPool) initialConnections(count int) {
	if count <= 0 {
		return
	}
	go func() {
		mu := sync.Mutex{}
		connections := make([]*PoolConn, 0, count)
		wg := sync.WaitGroup{}
		wg.Add(count)
		for i := 0; i < count; i++ {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
				defer cancel()
				conn, err := p.get(ctx, true)
				if err != nil {
					wg.Done()
					return
				}
				mu.Lock()
				connections = append(connections, conn)
				mu.Unlock()
				wg.Done()
			}()
		}
		wg.Wait()
		for _, conn := range connections {
			conn.Close()
		}
	}()
}

// Get 从连接池中获取连接
func (p *ConnectionPool) Get(ctx context.Context) (*PoolConn, error) {
	var (
		pc  *PoolConn
		err error
	)
	if pc, err = p.get(ctx, false); err != nil {
		report.ConnectionPoolGetConnectionErr.Incr()
		return nil, err
	}
	return pc, nil
}

// Close 释放连接
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.active -= p.idle.count
	pc := p.idle.head
	p.idle.count = 0
	p.idle.head, p.idle.tail = nil, nil
	if p.ch != nil {
		close(p.ch)
	}
	p.mu.Unlock()
	for ; pc != nil; pc = pc.next {
		pc.Conn.Close()
		pc.closed = true
	}
	return nil
}

// initializeCh 达到连接数上限后如果需要阻塞，需初始化p.ch用来同步
func (p *ConnectionPool) initializeCh() {
	p.initChOnce.Do(
		func() {
			p.ch = make(chan struct{}, p.MaxActive)
			if p.closed {
				close(p.ch)
			} else {
				for i := 0; i < p.MaxActive; i++ {
					p.ch <- struct{}{}
				}
			}
		},
	)
}

// get 从连接池获取连接
func (p *ConnectionPool) get(ctx context.Context, forceNew bool) (*PoolConn, error) {
	if p.Wait && p.MaxActive > 0 {
		p.initializeCh()
		if ctx == nil {
			<-p.ch
		} else {
			select {
			case <-p.ch:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	if !forceNew {
		// 尝试获取空闲连接
		if pc := p.getIdleConn(); pc != nil {
			return pc, nil
		}
	}

	// 获取新连接
	return p.getNewConn(ctx)
}

func (p *ConnectionPool) getIdleConn() *PoolConn {
	p.mu.Lock()
	for p.idle.head != nil {
		pc := p.idle.head
		p.idle.popHead()
		p.mu.Unlock()
		if p.checker(pc, true) {
			return pc
		}
		pc.Conn.Close()
		pc.closed = true
		p.mu.Lock()
		p.active--
	}
	p.mu.Unlock()
	return nil
}

func (p *ConnectionPool) getNewConn(ctx context.Context) (*PoolConn, error) {
	// 如果连接池已经关闭，则直接返回错误
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}

	// 检测是否超过连接池限制
	if p.overLimit() {
		p.mu.Unlock()
		return nil, ErrPoolLimit
	}

	p.active++
	p.mu.Unlock()
	c, err := p.dial(ctx)
	if err != nil {
		c = nil
		p.mu.Lock()
		p.active--
		if p.ch != nil && !p.closed {
			p.ch <- struct{}{}
		}
		p.mu.Unlock()
		return nil, err
	}
	report.ConnectionPoolGetNewConnection.Incr()
	return p.newPoolConn(c), nil
}

func (p *ConnectionPool) newPoolConn(c net.Conn) *PoolConn {
	pc := &PoolConn{
		Conn:       c,
		created:    time.Now(),
		pool:       p,
		forceClose: p.forceClosed,
	}
	if p.framerBuilder != nil {
		pc.fr = p.framerBuilder.New(codec.NewReader(pc))
		pc.copyFrame = !codec.IsSafeFramer(pc.fr)
	}
	return pc
}

func (p *ConnectionPool) checkHealthOnce() {
	p.mu.Lock()
	n := p.idle.count
	for i := 0; i < n && p.idle.head != nil; i++ {
		pc := p.idle.head
		p.idle.popHead()
		p.mu.Unlock()
		if p.checker(pc, false) {
			p.mu.Lock()
			p.idle.pushTail(pc)
		} else {
			p.mu.Lock()
			pc.Conn.Close()
			pc.closed = true
			p.active--
		}
	}
	p.mu.Unlock()
}

func (p *ConnectionPool) checkRoutine(interval time.Duration) {
	for {
		time.Sleep(interval)
		p.mu.Lock()
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return
		}
		p.checkHealthOnce()

		// 检查最小空闲连接数是否满足
		p.checkMinIdle()
	}
}

func (p *ConnectionPool) checkMinIdle() {
	if p.MinIdle <= 0 {
		return
	}
	p.mu.Lock()
	idle := p.idle.count
	p.mu.Unlock()
	if idle < p.MinIdle {
		p.initialConnections(p.MinIdle - idle)
	}
}

// RegisterChecker 注册空闲连接检查方法
func (p *ConnectionPool) RegisterChecker(interval time.Duration, checker healthChecker) {
	if interval <= 0 || checker == nil {
		return
	}
	p.mu.Lock()
	p.checker = checker
	p.mu.Unlock()
	go p.checkRoutine(interval)
}

// defaultChecker 默认空闲连接检查方法，返回 true 表示连接正常可用
func (p *ConnectionPool) defaultChecker(pc *PoolConn, isFast bool) bool {
	// 检查连接状态是否异常：已关闭，网络异常或者黏包处理异常
	if pc.isRemoteError(isFast) {
		return false
	}
	// 基于性能考虑，快速检查只做RemoteErr检查。
	if isFast {
		return true
	}
	// 检查连接是否已经超过最大空闲时间，如果是则关闭连接
	if p.IdleTimeout > 0 && pc.t.Add(p.IdleTimeout).Before(time.Now()) {
		report.ConnectionPoolIdleTimeout.Incr()
		return false
	}
	// 检查连接是否还处在生命周期内
	if p.MaxConnLifetime > 0 && pc.created.Add(p.MaxConnLifetime).Before(time.Now()) {
		report.ConnectionPoolLifetimeExceed.Incr()
		return false
	}
	return true
}

// overLimit 目前活跃连接数量大于最大限制，如果 Wait = false 则直接返回错误
func (p *ConnectionPool) overLimit() bool {
	if !p.Wait && p.MaxActive > 0 && p.active >= p.MaxActive {
		report.ConnectionPoolOverLimit.Incr()
		return true
	}
	return false
}

// dial 建立连接
func (p *ConnectionPool) dial(ctx context.Context) (net.Conn, error) {
	if p.Dial != nil {
		return p.Dial(ctx)
	}
	return nil, errors.New("must pass Dial to pool")
}

// put 尝试释放连接到连接池
func (p *ConnectionPool) put(pc *PoolConn, forceClose bool) error {
	if pc.closed {
		return nil
	}
	p.mu.Lock()
	if pc.closed {
		p.mu.Unlock()
		return nil
	}
	if !p.closed && !forceClose {
		pc.t = time.Now()
		p.idle.pushHead(pc)
		if p.idle.count > p.MaxIdle {
			pc = p.idle.tail
			p.idle.popTail()
		} else {
			pc = nil
		}
	}
	if pc != nil {
		p.mu.Unlock()
		pc.closed = true
		pc.Conn.Close()
		p.mu.Lock()
		p.active--
	}
	if p.ch != nil && !p.closed {
		p.ch <- struct{}{}
	}
	p.mu.Unlock()
	return nil
}

// PoolConn 连接池中的连接
type PoolConn struct {
	net.Conn
	fr         codec.Framer
	t          time.Time
	created    time.Time
	next, prev *PoolConn
	pool       *ConnectionPool
	closed     bool
	forceClose bool
	copyFrame  bool
}

// ReadFrame 读取帧
func (pc *PoolConn) ReadFrame() ([]byte, error) {
	if pc.closed {
		return nil, ErrConnClosed
	}
	if pc.fr == nil {
		pc.pool.put(pc, true)
		return nil, errors.New("framer not set")
	}
	data, err := pc.fr.ReadFrame()
	if err != nil {
		// ReadFrame 失败可能是socket Read接口超时失败
		// 或者解包失败，两种情况都应该关闭连接。
		pc.pool.put(pc, true)
		return nil, err
	}

	// Framer不支持并发读安全，拷贝一份数据
	if pc.copyFrame {
		buf := make([]byte, len(data))
		copy(buf, data)
		return buf, err
	}
	return data, err
}

// isRemoteError 尝试接收一个字节，探测对端是否已经主动关闭连接
// 对端返回 io.EOF 错误，表示对端已经关闭，
// 空闲的连接不应该读出数据，如果读出数据说明上层
// 没有做好黏包处理，也应该丢弃该连接
// 返回 true 表示连接有错误
func (pc *PoolConn) isRemoteError(isFast bool) bool {
	var err error
	if isFast {
		err = checkConnErrUnblock(pc.Conn, globalBuffer)
	} else {
		err = checkConnErr(pc.Conn, globalBuffer)
	}
	if err != nil {
		report.ConnectionPoolRemoteErr.Incr()
		return true
	}
	return false
}

// reset 重置连接状态
func (pc *PoolConn) reset() {
	if pc == nil {
		return
	}
	pc.Conn.SetDeadline(time.Time{})
}

// Write 连接上发送数据
func (pc *PoolConn) Write(b []byte) (int, error) {
	if pc.closed {
		return 0, ErrConnClosed
	}
	n, err := pc.Conn.Write(b)
	if err != nil {
		pc.pool.put(pc, true)
	}
	return n, err
}

// Read 连接上读取数据
func (pc *PoolConn) Read(b []byte) (int, error) {
	if pc.closed {
		return 0, ErrConnClosed
	}
	n, err := pc.Conn.Read(b)
	if err != nil {
		pc.pool.put(pc, true)
	}
	return n, err
}

// Close 重写net.Conn的Close方法，放回连接池
func (pc *PoolConn) Close() error {
	if pc.closed {
		return ErrConnClosed
	}
	pc.reset()
	return pc.pool.put(pc, pc.forceClose)
}

// connList 维护空闲连接，以栈的方式来维护连接
//
// 栈的方式相对队列有个好处，在请求量比较小但是请求分布仍比较均匀的情况下，队列方式会导致占用的连接迟迟得不到释放
type connList struct {
	count      int
	head, tail *PoolConn
}

func (l *connList) pushHead(pc *PoolConn) {
	pc.next = l.head
	pc.prev = nil
	if l.count == 0 {
		l.tail = pc
	} else {
		l.head.prev = pc
	}
	l.count++
	l.head = pc
}

func (l *connList) popHead() {
	pc := l.head
	l.count--
	if l.count == 0 {
		l.head, l.tail = nil, nil
	} else {
		pc.next.prev = nil
		l.head = pc.next
	}
	pc.next, pc.prev = nil, nil
}

func (l *connList) pushTail(pc *PoolConn) {
	pc.next = nil
	pc.prev = l.tail
	if l.count == 0 {
		l.head = pc
	} else {
		l.tail.next = pc
	}
	l.count++
	l.tail = pc
}

func (l *connList) popTail() {
	pc := l.tail
	l.count--
	if l.count == 0 {
		l.head, l.tail = nil, nil
	} else {
		pc.prev.next = nil
		l.tail = pc.prev
	}
	pc.next, pc.prev = nil, nil
}
