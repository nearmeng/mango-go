// Package writev 提供Buffer，使用writev()系统调用发包
package writev

import (
	"errors"
	"io"
	"net"
	"runtime"

	"git.code.oa.com/trpc-go/trpc-go/internal/ring"
	"git.code.oa.com/trpc-go/trpc-go/log"
)

const (
	// 默认缓冲队列长度
	defaultBufferSize = 128
	// writev最大可发送的数据包个数（来源Go源码定义）
	maxWritevBuffers = 1024
)

var (
	// ErrAskQuit 外部发送关闭请求
	ErrAskQuit = errors.New("writev goroutine is asked to quit")
	// ErrStopped Buffer停止接收数据
	ErrStopped = errors.New("writev buffer stop to receive data")
)

// QuitHandler Buffer协程退出处理函数
type QuitHandler func(*Buffer)

// Buffer 记录待发送报文，使用协程批量发送
type Buffer struct {
	opts           *Options      // 配置项
	w              io.Writer     // 底层发送数据的io.Writer
	queue          *ring.Ring    // 缓存报文的队列
	wakeupCh       chan struct{} // 用于唤醒发包协程
	done           chan struct{} // 通知发包协程退出
	err            error         // 记录错误信息
	errCh          chan error    // 内部错误通知
	isQueueStopped bool          // 缓存队列是否停止收包
}

var defaultQuitHandler = func(b *Buffer) {
	b.SetQueueStopped(true)
}

// NewBuffer 创建Buffer，并启动发送协程
func NewBuffer(opt ...Option) *Buffer {
	opts := &Options{
		bufferSize: defaultBufferSize,
		handler:    defaultQuitHandler,
	}
	for _, o := range opt {
		o(opts)
	}

	b := &Buffer{
		queue:    ring.New(uint32(opts.bufferSize)),
		opts:     opts,
		wakeupCh: make(chan struct{}, 1),
		errCh:    make(chan error, 1),
	}
	return b
}

// Start 启动发送协程，需要在启动时设置writer和done
func (b *Buffer) Start(writer io.Writer, done chan struct{}) {
	b.w = writer
	b.done = done
	go b.start()
}

// Restart 重启时重新创建一个Buffer，复用原始Buffer的缓冲队列和配置
func (b *Buffer) Restart(writer io.Writer, done chan struct{}) *Buffer {
	buffer := &Buffer{
		queue:    b.queue,
		opts:     b.opts,
		wakeupCh: make(chan struct{}, 1),
		errCh:    make(chan error, 1),
	}
	buffer.Start(writer, done)
	return buffer
}

// SetQueueStopped 设置缓冲队列是否停止收包
func (b *Buffer) SetQueueStopped(stopped bool) {
	b.isQueueStopped = stopped
	if b.err == nil {
		b.err = ErrStopped
	}
}

// Write 把p写入缓冲队列，返回写入的数据长度。
// 如何写包小于len(p), err返回具体原因
func (b *Buffer) Write(p []byte) (int, error) {
	if b.opts.dropFull {
		return b.writeNoWait(p)
	}
	return b.writeOrWait(p)
}

// Error 返回发送协程退出的原因
func (b *Buffer) Error() error {
	return b.err
}

// Done 返回退出Channel
func (b *Buffer) Done() chan struct{} {
	return b.done
}

func (b *Buffer) wakeUp() {
	// 基于性能优化考虑：由于并发select写channel效率较差，此处优先检查
	// wakeupCh是否已经有唤醒消息，减少并发写channel的几率
	if len(b.wakeupCh) > 0 {
		return
	}
	// 尝试发送唤醒信号，不等待
	select {
	case b.wakeupCh <- struct{}{}:
	default:
	}
}

func (b *Buffer) writeNoWait(p []byte) (int, error) {
	// 缓冲队列停止收包，直接返回
	if b.isQueueStopped {
		return 0, b.err
	}
	// 队列满时直接返回
	if err := b.queue.Put(p); err != nil {
		return 0, err
	}
	// 写缓冲队列成功，唤醒发包协程
	b.wakeUp()
	return len(p), nil
}

func (b *Buffer) writeOrWait(p []byte) (int, error) {
	for {
		// 缓冲队列停止收包，直接返回
		if b.isQueueStopped {
			return 0, b.err
		}
		// 写缓冲队列成功，唤醒发包协程
		if err := b.queue.Put(p); err == nil {
			b.wakeUp()
			return len(p), nil
		}
		// 队列已满，直接发包
		if err := b.writeDirectly(); err != nil {
			return 0, err
		}
	}
}

func (b *Buffer) writeDirectly() error {
	if b.queue.IsEmpty() {
		return nil
	}
	vals := make([]interface{}, 0, maxWritevBuffers)
	size, _ := b.queue.Gets(&vals)
	if size == 0 {
		return nil
	}
	bufs := make(net.Buffers, 0, maxWritevBuffers)
	for i := range vals {
		buf, ok := vals[i].([]byte)
		if !ok {
			log.Tracef("writev: buffer that get from ring is not []byte type.")
			continue
		}
		bufs = append(bufs, buf)
	}
	if _, err := bufs.WriteTo(b.w); err != nil {
		// 通知发包协程设置错误并退出
		select {
		case b.errCh <- err:
		default:
		}
		return err
	}
	return nil
}

func (b *Buffer) getOrWait(values *[]interface{}) error {
	for {
		// 检查是否被通知关闭发包协程
		select {
		case <-b.done:
			return ErrAskQuit
		case err := <-b.errCh:
			return err
		default:
		}
		// 从缓存队列批量收包
		size, _ := b.queue.Gets(values)
		if size > 0 {
			return nil
		}

		// Fast Path：由于通过采用select唤醒协程的性能较差，这里优先使用
		// Gosched()延迟检查队列，在较高负荷场景下提升队列有包的命中率和
		// 批量获取包的效率, 进而降低使用select唤醒协程的几率。
		runtime.Gosched()
		if !b.queue.IsEmpty() {
			continue
		}
		// Slow Path：延迟检查队列之后仍然没有包，说明系统比较空闲。协程通过
		// select机制等待唤醒。休眠的好处在于在系统空闲状态下减少CPU空转损耗。
		select {
		case <-b.done:
			return ErrAskQuit
		case err := <-b.errCh:
			return err
		case <-b.wakeupCh:
		}
	}
}

func (b *Buffer) start() {
	initBufs := make(net.Buffers, 0, maxWritevBuffers)
	vals := make([]interface{}, 0, maxWritevBuffers)
	bufs := initBufs

	defer b.opts.handler(b)
	for {
		if err := b.getOrWait(&vals); err != nil {
			b.err = err
			break
		}

		for i := range vals {
			buf, ok := vals[i].([]byte)
			if !ok {
				log.Tracef("writev: buffer that get from ring is not []byte type.")
				continue
			}
			bufs = append(bufs, buf)
		}
		vals = vals[:0]

		if _, err := bufs.WriteTo(b.w); err != nil {
			b.err = err
			break
		}
		// 重置bufs到初始位置，防止append产生新的内存分配
		bufs = initBufs
	}
}
