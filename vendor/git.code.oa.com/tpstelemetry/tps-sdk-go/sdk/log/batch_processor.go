// Copyright 2020 The TpsTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package log log 组件
package log

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	logsproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/logs/v1"
	v1proto "github.com/golang/protobuf/proto"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/pkg/metrics"

	"go.opentelemetry.io/otel"
)

const (
	DefaultMaxQueueSize         = 2048
	DefaultBatchTimeout         = 5000 * time.Millisecond
	DefaultMaxExportBatchSize   = 512
	DefaultMaxBatchedPacketSize = 2097152
)

var (
	failedExportCounter      = metrics.BatchProcessCounter.WithLabelValues("failed", "logs")
	succeededExportCounter   = metrics.BatchProcessCounter.WithLabelValues("success", "logs")
	batchByCountCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchcount")
	batchByPacketSizeCounter = metrics.BatchProcessCounter.WithLabelValues("batched", "packetsize")
	batchByTimerCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchtimer")
	enqueueCounter           = metrics.BatchProcessCounter.WithLabelValues("enqueue", "logs")
	dropCounter              = metrics.BatchProcessCounter.WithLabelValues("dropped", "logs")
)

// BatchLogProcessorOption BatchLogProcessor Option helper
type BatchLogProcessorOption func(o *BatchLogProcessorOptions)

// BatchLogProcessorOptions BatchLogProcessor 控制项
type BatchLogProcessorOptions struct {
	// MaxQueueSize is the maximum queue size to buffer logs for delayed processing. If the
	// queue gets full it drops the logs. Use BlockOnQueueFull to change this behavior.
	// The default value of MaxQueueSize is 2048.
	MaxQueueSize int

	// BatchTimeout is the maximum duration for constructing a batch. Processor
	// forcefully sends available logs when timeout is reached.
	// The default value of BatchTimeout is 5000 msec.
	BatchTimeout time.Duration

	// MaxExportBatchSize is the maximum number of logs to process in a single batch.
	// If there are more than one batch worth of logs then it processes multiple batches
	// of logs one batch after the other without any delay.
	// The default value of MaxExportBatchSize is 512.
	MaxExportBatchSize int

	// MaxPacketSize is the maximum number of packet size that will forcefully trigger a batch process.
	// The deault value of MaxPacketSize is 2M (in bytes) .
	MaxPacketSize int

	// BlockOnQueueFull blocks onEnd() and onStart() method if the queue is full
	// AND if BlockOnQueueFull is set to true.
	// Blocking option should be used carefully as it can severely affect the performance of an
	// application.
	BlockOnQueueFull bool
}

// WithMaxQueueSize 设置queuesize 选项
func WithMaxQueueSize(size int) BatchLogProcessorOption {
	return func(o *BatchLogProcessorOptions) {
		o.MaxQueueSize = size
	}
}

// WithMaxExportBatchSize 设置 MaxExportBatchSize 选项
func WithMaxExportBatchSize(size int) BatchLogProcessorOption {
	return func(o *BatchLogProcessorOptions) {
		o.MaxExportBatchSize = size
	}
}

// WithBatchTimeout 设置BatchTimeout 选项
func WithBatchTimeout(delay time.Duration) BatchLogProcessorOption {
	return func(o *BatchLogProcessorOptions) {
		o.BatchTimeout = delay
	}
}

// WithBlocking 设置blocking 选项
func WithBlocking() BatchLogProcessorOption {
	return func(o *BatchLogProcessorOptions) {
		o.BlockOnQueueFull = true
	}
}

// WithMaxPacketSize 这只最大包 选项
func WithMaxPacketSize(size int) BatchLogProcessorOption {
	return func(o *BatchLogProcessorOptions) {
		o.MaxPacketSize = size
	}
}

// BatchProcessor 批处理任务单元
type BatchProcessor struct {
	e Exporter
	o BatchLogProcessorOptions

	queue       chan *logsproto.ResourceLogs
	dropped     uint32
	batchedSize int

	batch      []*logsproto.ResourceLogs
	batchMutex sync.Mutex
	timer      *time.Timer
	stopWait   sync.WaitGroup
	stopOnce   sync.Once
	stopCh     chan struct{}
}

// NewBatchProcessor BatchProcessor 工厂方法
func NewBatchProcessor(exporter Exporter, options ...BatchLogProcessorOption) *BatchProcessor {
	o := BatchLogProcessorOptions{
		BatchTimeout:       DefaultBatchTimeout,
		MaxQueueSize:       DefaultMaxQueueSize,
		MaxExportBatchSize: DefaultMaxExportBatchSize,
		MaxPacketSize:      DefaultMaxBatchedPacketSize,
	}
	for _, opt := range options {
		opt(&o)
	}

	bp := &BatchProcessor{
		e:      exporter,
		o:      o,
		batch:  make([]*logsproto.ResourceLogs, 0, o.MaxExportBatchSize),
		timer:  time.NewTimer(o.BatchTimeout),
		queue:  make(chan *logsproto.ResourceLogs, o.MaxQueueSize),
		stopCh: make(chan struct{}),
	}

	bp.stopWait.Add(1)
	go func() {
		defer bp.stopWait.Done()
		bp.processQueue()
		bp.drainQueue()
	}()

	return bp
}

// Shutdown flushes the queue and waits until all logs are processed.
// It only executes once. Subsequent call does nothing.
func (bp *BatchProcessor) Shutdown(ctx context.Context) error {
	var err error
	bp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			close(bp.stopCh)
			bp.stopWait.Wait()
			if bp.e != nil {
				if e := bp.e.Shutdown(ctx); e != nil {
					otel.Handle(err)
				}
			}
			close(wait)
		}()
		// Wait until the wait group is done or the context is cancelled
		select {
		case <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}

// ForceFlush exports all ended logs that have not yet been exported.
func (bp *BatchProcessor) ForceFlush(ctx context.Context) error {
	return bp.exportLogs(ctx)
}

// exportLogs is a subroutine of processing and draining the queue.
func (bp *BatchProcessor) exportLogs(ctx context.Context) error {
	bp.timer.Reset(bp.o.BatchTimeout)

	bp.batchMutex.Lock()
	defer bp.batchMutex.Unlock()

	if len(bp.batch) > 0 {
		size := len(bp.batch)
		err := bp.e.ExportLogs(ctx, bp.batch)
		bp.batch = bp.batch[:0]
		bp.batchedSize = 0
		if err != nil {
			failedExportCounter.Add(float64(size))
			return err
		}
		succeededExportCounter.Add(float64(size))
	}
	return nil
}

// processQueue removes logs from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bp *BatchProcessor) processQueue() {
	defer bp.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-bp.stopCh:
			return
		case <-bp.timer.C:
			batchByTimerCounter.Inc()
			if err := bp.exportLogs(ctx); err != nil {
				otel.Handle(err)
			}
		case rl := <-bp.queue:
			bp.batchMutex.Lock()
			bp.batch = append(bp.batch, rl)
			bp.batchedSize += calcLogSize(rl)
			shouldExport := bp.shouldProcessInBatch()
			bp.batchMutex.Unlock()
			if shouldExport {
				if !bp.timer.Stop() {
					<-bp.timer.C
				}
				if err := bp.exportLogs(ctx); err != nil {
					otel.Handle(err)
				}
			}
		}
	}
}

// drainQueue awaits the any caller that had added to bp.stopWait
// to finish the enqueue, then exports the final batch.
func (bp *BatchProcessor) drainQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case rl := <-bp.queue:
			if rl == nil {
				if err := bp.exportLogs(ctx); err != nil {
					otel.Handle(err)
				}
				return
			}

			bp.batchMutex.Lock()
			bp.batch = append(bp.batch, rl)
			bp.batchedSize += calcLogSize(rl)
			shouldExport := bp.shouldProcessInBatch()
			bp.batchMutex.Unlock()

			if shouldExport {
				if err := bp.exportLogs(ctx); err != nil {
					otel.Handle(err)
				}
			}
		default:
			close(bp.queue)
		}
	}
}

// shouldProcessInBatch determines whether to export in batches
func (bp *BatchProcessor) shouldProcessInBatch() bool {
	if len(bp.batch) == bp.o.MaxExportBatchSize {
		batchByCountCounter.Inc()
		return true
	}
	if bp.batchedSize >= bp.o.MaxPacketSize {
		batchByPacketSizeCounter.Inc()
		return true
	}
	return false
}

// Enqueue 日志入队
func (bp *BatchProcessor) Enqueue(rl *logsproto.ResourceLogs) {
	enqueueCounter.Inc()

	// This ensures the bp.queue<- below does not panic as the
	// processor shuts down.
	defer func() {
		x := recover()
		switch err := x.(type) {
		case nil:
			return
		case runtime.Error:
			if err.Error() == "send on closed channel" {
				return
			}
		}
		panic(x)
	}()

	select {
	case <-bp.stopCh:
		return
	default:
	}

	if bp.o.BlockOnQueueFull {
		bp.queue <- rl
		return
	}

	select {
	case bp.queue <- rl:
	default:
		atomic.AddUint32(&bp.dropped, 1)
		dropCounter.Inc()
	}
}

// calcLogSize calculates the packet size of a ResourceLogs
func calcLogSize(rl *logsproto.ResourceLogs) int {
	return v1proto.Size(rl)
}
