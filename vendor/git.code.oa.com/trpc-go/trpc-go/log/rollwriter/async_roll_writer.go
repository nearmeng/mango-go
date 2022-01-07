package rollwriter

import (
	"bytes"
	"errors"
	"io"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/internal/report"
)

// AsyncRollWriter 日志异步写入类
// 实现zap.WriteSyncer接口
type AsyncRollWriter struct {
	logger io.Writer
	opts   *AsyncOptions

	logQueue chan []byte
	syncChan chan struct{}
}

// NewAsyncRollWriter 根据传入的参数创建一个RollWriter对
func NewAsyncRollWriter(logger io.Writer, opt ...AsyncOption) *AsyncRollWriter {
	opts := &AsyncOptions{
		LogQueueSize:     10000,    // 默认队列长度为10000
		WriteLogSize:     4 * 1024, // 默认4k刷盘一次
		WriteLogInterval: 100,      // 默认100ms刷盘一次
		DropLog:          false,    // 默认不丢弃日志
	}

	// 输入参数为最高优先级 覆盖掉原有数据
	for _, o := range opt {
		o(opts)
	}

	w := &AsyncRollWriter{}
	w.logger = logger
	w.opts = opts
	w.logQueue = make(chan []byte, opts.LogQueueSize)
	w.syncChan = make(chan struct{})

	// 起一个协程，批量异步写入日志
	go w.batchWriteLog()
	return w
}

// Write 写日志
// 实现io.Writer
func (w *AsyncRollWriter) Write(data []byte) (int, error) {
	log := make([]byte, len(data))
	copy(log, data)
	if w.opts.DropLog {
		select {
		case w.logQueue <- log:
		default:
			report.LogQueueDropNum.Incr()
			return 0, errors.New("log queue is full")
		}
	} else {
		w.logQueue <- log
	}
	return len(data), nil
}

// Sync 同步日志
// 实现zap.WriteSyncer接口
func (w *AsyncRollWriter) Sync() error {
	w.syncChan <- struct{}{}
	return nil
}

// Close 关闭当前日志文件
// 实现io.Closer
func (w *AsyncRollWriter) Close() error {
	return w.Sync()
}

// batchWriteLog 批量异步写入日志
func (w *AsyncRollWriter) batchWriteLog() {
	buffer := bytes.NewBuffer(make([]byte, 0, w.opts.WriteLogSize*2))
	ticker := time.NewTicker(time.Millisecond * time.Duration(w.opts.WriteLogInterval))
	for {
		select {
		case <-ticker.C:
			if buffer.Len() > 0 {
				_, _ = w.logger.Write(buffer.Bytes())
				buffer.Reset()
			}
		case data := <-w.logQueue:
			buffer.Write(data)
			if buffer.Len() >= w.opts.WriteLogSize {
				_, _ = w.logger.Write(buffer.Bytes())
				buffer.Reset()
			}
		case <-w.syncChan:
			if buffer.Len() > 0 {
				_, _ = w.logger.Write(buffer.Bytes())
				buffer.Reset()
			}
			size := len(w.logQueue)
			for i := 0; i < size; i++ {
				v := <-w.logQueue
				_, _ = w.logger.Write(v)
			}
		}
	}
}
