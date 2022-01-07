// Copyright 2021 The TpsTelemetry Authors
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

// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package trace trace 组件
package trace // import "go.opentelemetry.io/otel/sdk/trace"

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/pkg/metrics"
)

var (
	DefaultMaxQueueSize         = 2048                    // DefaultMaxQueueSize 默认最大queque size
	DefaultBatchTimeout         = 5000 * time.Millisecond // DefaultBatchTimeout 默认批处理定时时间
	DefaultMaxExportBatchSize   = 512                     // DefaultMaxExportBatchSize 默认最大上传批大小
	DefaultMaxBatchedPacketSize = 2097152                 // DefaultMaxBatchedPacketSize 触发大包上传默认值
)

var (
	failedExportCounter      = metrics.BatchProcessCounter.WithLabelValues("failed", "traces")
	succeededExportCounter   = metrics.BatchProcessCounter.WithLabelValues("success", "traces")
	batchByCountCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchcount")
	batchByPacketSizeCounter = metrics.BatchProcessCounter.WithLabelValues("batched", "packetsize")
	batchByTimerCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchtimer")
	enqueueCounter           = metrics.BatchProcessCounter.WithLabelValues("enqueue", "traces")
	dropCounter              = metrics.BatchProcessCounter.WithLabelValues("dropped", "traces")
)

// BatchSpanProcessorOption BatchSpanProcessor Option helper
type BatchSpanProcessorOption func(o *BatchSpanProcessorOptions)

// BatchSpanProcessorOptions BatchSpanProcessor 控制项
type BatchSpanProcessorOptions struct {
	// MaxQueueSize is the maximum queue size to buffer spans for delayed processing. If the
	// queue gets full it drops the spans. Use BlockOnQueueFull to change this behavior.
	// The default value of MaxQueueSize is 2048.
	MaxQueueSize int

	// BatchTimeout is the maximum duration for constructing a batch. Processor
	// forcefully sends available spans when timeout is reached.
	// The default value of BatchTimeout is 5000 msec.
	BatchTimeout time.Duration

	// MaxExportBatchSize is the maximum number of spans to process in a single batch.
	// If there are more than one batch worth of spans then it processes multiple batches
	// of spans one batch after the other without any delay.
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

// batchSpanProcessor is a SpanProcessor that batches asynchronously-received
// SpanSnapshots and sends them to a trace.Exporter when complete.
type batchSpanProcessor struct {
	e export.SpanExporter
	o BatchSpanProcessorOptions

	queue       chan *export.SpanSnapshot
	dropped     uint32
	batchedSize int

	batch      []*export.SpanSnapshot
	batchMutex sync.Mutex
	timer      *time.Timer
	stopWait   sync.WaitGroup
	stopOnce   sync.Once
	stopCh     chan struct{}
}

var _ sdktrace.SpanProcessor = (*batchSpanProcessor)(nil)

// NewBatchSpanProcessor creates a new SpanProcessor that will send completed
// span batches to the exporter with the supplied options.
//
// If the exporter is nil, the span processor will preform no action.
func NewBatchSpanProcessor(exporter export.SpanExporter, options ...BatchSpanProcessorOption) sdktrace.SpanProcessor {
	o := BatchSpanProcessorOptions{
		BatchTimeout:       DefaultBatchTimeout,
		MaxQueueSize:       DefaultMaxQueueSize,
		MaxExportBatchSize: DefaultMaxExportBatchSize,
		MaxPacketSize:      DefaultMaxBatchedPacketSize,
	}
	for _, opt := range options {
		opt(&o)
	}
	bsp := &batchSpanProcessor{
		e:      exporter,
		o:      o,
		batch:  make([]*export.SpanSnapshot, 0, o.MaxExportBatchSize),
		timer:  time.NewTimer(o.BatchTimeout),
		queue:  make(chan *export.SpanSnapshot, o.MaxQueueSize),
		stopCh: make(chan struct{}),
	}

	bsp.stopWait.Add(1)
	go func() {
		defer bsp.stopWait.Done()
		bsp.processQueue()
		bsp.drainQueue()
	}()

	return bsp
}

// OnStart method does nothing.
func (bsp *batchSpanProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {}

// OnEnd method enqueues a ReadOnlySpan for later processing.
func (bsp *batchSpanProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	// Do not enqueue spans if we are just going to drop them.
	if bsp.e == nil {
		return
	}
	bsp.enqueue(s.Snapshot())
}

// Shutdown flushes the queue and waits until all spans are processed.
// It only executes once. Subsequent call does nothing.
func (bsp *batchSpanProcessor) Shutdown(ctx context.Context) error {
	var err error
	bsp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			close(bsp.stopCh)
			bsp.stopWait.Wait()
			if bsp.e != nil {
				if e := bsp.e.Shutdown(ctx); e != nil {
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

// ForceFlush exports all ended spans that have not yet been exported.
func (bsp *batchSpanProcessor) ForceFlush(ctx context.Context) error {
	return bsp.exportSpans(ctx)
}

// WithMaxQueueSize 设置 MaxQueueSize helper
func WithMaxQueueSize(size int) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.MaxQueueSize = size
	}
}

// WithMaxExportBatchSize 设置 MaxExportBatchSize helper
func WithMaxExportBatchSize(size int) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.MaxExportBatchSize = size
	}
}

// WithBatchTimeout 设置 BatchTimeout helper
func WithBatchTimeout(delay time.Duration) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.BatchTimeout = delay
	}
}

// WithBlocking 设置 Blocking helper
func WithBlocking() BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.BlockOnQueueFull = true
	}
}

// WithMaxPacketSize 设置 MaxPacketSize helper
func WithMaxPacketSize(size int) BatchSpanProcessorOption {
	return func(o *BatchSpanProcessorOptions) {
		o.MaxPacketSize = size
	}
}

// exportSpans is a subroutine of processing and draining the queue.
func (bsp *batchSpanProcessor) exportSpans(ctx context.Context) error {
	bsp.timer.Reset(bsp.o.BatchTimeout)

	bsp.batchMutex.Lock()
	defer bsp.batchMutex.Unlock()

	if len(bsp.batch) > 0 {
		size := len(bsp.batch)
		err := bsp.e.ExportSpans(ctx, bsp.batch)
		bsp.batch = bsp.batch[:0]
		bsp.batchedSize = 0
		if err != nil {
			failedExportCounter.Add(float64(size))
			return err
		}
		succeededExportCounter.Add(float64(size))
	}
	return nil
}

// processQueue removes spans from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bsp *batchSpanProcessor) processQueue() {
	defer bsp.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-bsp.stopCh:
			return
		case <-bsp.timer.C:
			batchByTimerCounter.Inc()
			if err := bsp.exportSpans(ctx); err != nil {
				otel.Handle(err)
			}
		case sd := <-bsp.queue:
			bsp.batchMutex.Lock()
			bsp.batch = append(bsp.batch, sd)
			bsp.batchedSize += calcSpanSize(sd)
			shouldExport := bsp.shouldProcessInBatch()
			bsp.batchMutex.Unlock()
			if shouldExport {
				if !bsp.timer.Stop() {
					<-bsp.timer.C
				}
				if err := bsp.exportSpans(ctx); err != nil {
					otel.Handle(err)
				}
			}
		}
	}
}

// drainQueue awaits the any caller that had added to bsp.stopWait
// to finish the enqueue, then exports the final batch.
func (bsp *batchSpanProcessor) drainQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case sd := <-bsp.queue:
			if sd == nil {
				if err := bsp.exportSpans(ctx); err != nil {
					otel.Handle(err)
				}
				return
			}

			bsp.batchMutex.Lock()
			bsp.batch = append(bsp.batch, sd)
			bsp.batchedSize += calcSpanSize(sd)
			shouldExport := bsp.shouldProcessInBatch()
			bsp.batchMutex.Unlock()

			if shouldExport {
				if err := bsp.exportSpans(ctx); err != nil {
					otel.Handle(err)
				}
			}
		default:
			close(bsp.queue)
		}
	}
}

// shouldProcessInBatch determines whether to export in batches
func (bsp *batchSpanProcessor) shouldProcessInBatch() bool {
	if len(bsp.batch) == bsp.o.MaxExportBatchSize {
		batchByCountCounter.Inc()
		return true
	}

	if bsp.batchedSize >= bsp.o.MaxPacketSize {
		batchByPacketSizeCounter.Inc()
		return true
	}

	return false
}

func (bsp *batchSpanProcessor) enqueue(sd *export.SpanSnapshot) {
	enqueueCounter.Inc()

	// This ensures the bsp.queue<- below does not panic as the
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
	case <-bsp.stopCh:
		return
	default:
	}

	if bsp.o.BlockOnQueueFull {
		bsp.queue <- sd
		return
	}

	select {
	case bsp.queue <- sd:
	default:
		atomic.AddUint32(&bsp.dropped, 1)
		dropCounter.Inc()
	}
}

// calcSpanSize calculates the packet size of a SpanSnapShot
func calcSpanSize(sd *export.SpanSnapshot) int {
	if sd == nil {
		return 0
	}

	size := 0
	// just calculate events size for now.
	for _, event := range sd.MessageEvents {
		for _, kv := range event.Attributes {
			size += len(kv.Key)
			size += len(kv.Value.AsString())
		}
	}
	return size
}
