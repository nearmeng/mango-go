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

// Package tpszap zap组件适配
package tpszap

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	commonproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/common/v1"
	logsproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/logs/v1"
	resourceproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/resource/v1"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/pkg/metrics"
	sdklog "git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/log"

	v1proto "github.com/golang/protobuf/proto"
	jsoniter "github.com/json-iterator/go"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.uber.org/zap/zapcore"
)

const (
	DefaultMaxQueueSize         = 2048
	DefaultBatchTimeout         = 5000 * time.Millisecond
	DefaultMaxExportBatchSize   = 512
	DefaultBlockOnQueueFull     = false
	DefaultMaxBatchedPacketSize = 2097152
)

var _ zapcore.WriteSyncer = (*BatchWriteSyncer)(nil)

var (
	failedExportCounter      = metrics.BatchProcessCounter.WithLabelValues("failed", "logs")
	succeededExportCounter   = metrics.BatchProcessCounter.WithLabelValues("success", "logs")
	batchByCountCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchcount")
	batchByPacketSizeCounter = metrics.BatchProcessCounter.WithLabelValues("batched", "packetsize")
	batchByTimerCounter      = metrics.BatchProcessCounter.WithLabelValues("batched", "batchtimer")
	enqueueCounter           = metrics.BatchProcessCounter.WithLabelValues("enqueue", "logs")
	dropCounter              = metrics.BatchProcessCounter.WithLabelValues("dropped", "logs")
)

// BatchWriteSyncer BatchWriteSyncer
type BatchWriteSyncer struct {
	exporter sdklog.Exporter
	opt      *BatchSyncerOptions

	queue       chan *logsproto.ResourceLogs
	dropped     uint32
	batch       []*logsproto.ResourceLogs
	timer       *time.Timer
	batchedSize int
	rs          *resource.Resource
	stopCh      chan struct{}
	rspb        *resourceproto.Resource
}

const (
	fieldSampled = "sampled"
	trueString   = "true"
)

// logsField ...
var logsField = map[string]bool{
	"msg":     true,
	"traceID": true,
	"spanID":  true,
	"sampled": true,
	"caller":  true,
	"level":   true,
	"ts":      true,
}

// isTagsField ...
func isTagsField(key string) bool {
	return !logsField[key]
}

func convertToRecord(raw map[string]interface{}) *logsproto.LogRecord {
	l := &logsproto.LogRecord{}
	msg, ok := raw["msg"]
	if !ok {
		return l
	}
	msgs, ok := msg.(string)
	if !ok {
		return l
	}
	l.Body = &commonproto.AnyValue{
		Value: &commonproto.AnyValue_StringValue{
			StringValue: msgs,
		},
	}
	traceID, ok := raw["traceID"]
	if ok {
		var traceIDStr string
		traceIDStr, _ = traceID.(string)
		l.TraceId, _ = hex.DecodeString(traceIDStr)
	}
	spanID, ok := raw["spanID"]
	if ok {
		var spanIDStr string
		spanIDStr, _ = spanID.(string)
		l.SpanId, _ = hex.DecodeString(spanIDStr)
	}
	sampledRaw, ok := raw[fieldSampled]
	if ok {
		sampledStr, _ := sampledRaw.(string)
		sampled := (sampledStr == trueString)
		l.Attributes = append(l.Attributes, &commonproto.KeyValue{
			Key: fieldSampled,
			Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_BoolValue{
				BoolValue: sampled,
			}},
		})
	}
	lineRaw, ok := raw["caller"]
	if ok {
		var line string
		line, ok = lineRaw.(string)
		if ok {
			l.Attributes = append(l.Attributes, &commonproto.KeyValue{
				Key: "line",
				Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_StringValue{
					StringValue: line,
				}},
			})
		}
	}
	levelRaw, ok := raw["level"]
	if ok {
		var level string
		level, ok = levelRaw.(string)
		if ok {
			l.SeverityText = level
		}
	}

	tsRaw, ok := raw["ts"]
	if ok {
		ts, ok := tsRaw.(float64)
		if ok {
			l.TimeUnixNano = uint64(ts * float64(time.Second))
		}
	}

	for k, v := range raw {
		if vv, ok := v.(string); ok && isTagsField(k) {
			l.Attributes = append(l.Attributes, &commonproto.KeyValue{
				Key: k,
				Value: &commonproto.AnyValue{Value: &commonproto.AnyValue_StringValue{
					StringValue: vv,
				}},
			})
		}
	}
	return l
}

// Write Write
func (bp *BatchWriteSyncer) Write(p []byte) (n int, err error) {
	raw := make(map[string]interface{})
	err = jsoniter.ConfigFastest.Unmarshal(p, &raw)
	if err != nil {
		return 0, err
	}
	if bp.opt.EnabelSampler { // 只上报采样的
		if sampled, _ := raw[fieldSampled].(string); sampled != trueString {
			return 0, nil
		}
	}

	l := convertToRecord(raw)

	rl := &logsproto.ResourceLogs{
		Resource: bp.rspb,
		InstrumentationLibraryLogs: []*logsproto.InstrumentationLibraryLogs{
			{
				Logs: []*logsproto.LogRecord{l},
			},
		},
	}
	bp.Enqueue(rl, 1)
	return len(p), nil
}

// Sync Sync
func (bp *BatchWriteSyncer) Sync() error {
	bp.export()
	return nil
}

// NewBatchWriteSyncer NewBatchWriteSyncer
func NewBatchWriteSyncer(exporter sdklog.Exporter, rs *resource.Resource, opts ...BatchSyncerOption) *BatchWriteSyncer {
	opt := &BatchSyncerOptions{
		MaxQueueSize:       DefaultMaxQueueSize,
		BatchTimeout:       DefaultBatchTimeout,
		MaxExportBatchSize: DefaultMaxExportBatchSize,
		BlockOnQueueFull:   DefaultBlockOnQueueFull,
		MaxPacketSize:      DefaultMaxBatchedPacketSize,
	}

	for _, o := range opts {
		o(opt)
	}

	bp := &BatchWriteSyncer{
		opt:      opt,
		rs:       rs,
		exporter: exporter,
		batch:    make([]*logsproto.ResourceLogs, 0, opt.MaxExportBatchSize),
		queue:    make(chan *logsproto.ResourceLogs, opt.MaxQueueSize),
		stopCh:   make(chan struct{}),
		timer:    time.NewTimer(opt.BatchTimeout),
	}
	if rs.Len() != 0 {
		rspb := &resourceproto.Resource{}
		for _, kv := range rs.Attributes() {
			rspb.Attributes = append(rspb.Attributes, &commonproto.KeyValue{
				Key: string(kv.Key),
				Value: &commonproto.AnyValue{
					Value: &commonproto.AnyValue_StringValue{StringValue: kv.Value.Emit()},
				},
			})
		}
		bp.rspb = rspb
	}

	go func() {
		bp.processQueue()
		bp.drainQueue()
	}()

	return bp
}

// Enqueue Enqueue
func (bp *BatchWriteSyncer) Enqueue(rl *logsproto.ResourceLogs, size int) {
	enqueueCounter.Add(float64(size))
	select {
	case <-bp.stopCh:
		return
	default:
	}

	if bp.opt.BlockOnQueueFull {
		bp.queue <- rl
		return
	}

	select {
	case bp.queue <- rl:
	default:
		dropCounter.Add(float64(size))
		otel.Handle(errors.New("tpstelemetry export logs dropped"))
		atomic.AddUint32(&bp.dropped, 1)
	}
}

func (bp *BatchWriteSyncer) processQueue() {
	defer bp.timer.Stop()

	for {
		select {
		case <-bp.stopCh:
			return
		case <-bp.timer.C:
			batchByTimerCounter.Inc()
			bp.export()
		case ld := <-bp.queue:
			bp.batch = append(bp.batch, ld)
			bp.batchedSize += calcLogSize(ld)
			shouldExport := bp.shouldProcessInBatch()
			if shouldExport {
				if !bp.timer.Stop() {
					<-bp.timer.C
				}
				bp.export()
			}
		}
	}
}

func (bp *BatchWriteSyncer) export() {
	bp.timer.Reset(bp.opt.BatchTimeout)
	if len(bp.batch) > 0 {
		size := len(bp.batch)
		err := bp.exporter.ExportLogs(context.Background(), bp.batch)
		bp.batch = bp.batch[:0]
		bp.batchedSize = 0
		if err != nil {
			otel.Handle(fmt.Errorf("tpstelemetry export logs failed: %v", err))
			failedExportCounter.Add(float64(size))
		} else {
			succeededExportCounter.Add(float64(size))
		}
	}
}

func (bp *BatchWriteSyncer) drainQueue() {
	for {
		select {
		case ld := <-bp.queue:
			if ld == nil {
				bp.export()
				return
			}
			bp.batch = append(bp.batch, ld)
			bp.batchedSize += calcLogSize(ld)
			shouldExport := bp.shouldProcessInBatch()
			if shouldExport {
				bp.export()
			}
		default:
			close(bp.queue)
		}
	}
}

// shouldProcessInBatch determines whether to export in batches
func (bp *BatchWriteSyncer) shouldProcessInBatch() bool {
	if len(bp.batch) == bp.opt.MaxExportBatchSize {
		batchByCountCounter.Inc()
		return true
	}
	if bp.batchedSize >= bp.opt.MaxPacketSize {
		batchByPacketSizeCounter.Inc()
		return true
	}
	return false
}

// BatchSyncerOption BatchSyncerOption
type BatchSyncerOption func(o *BatchSyncerOptions)

// BatchSyncerOptions BatchSyncerOptions
type BatchSyncerOptions struct {
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

	// BlockOnQueueFull blocks onEnd() and onStart() method if the queue is full
	// AND if BlockOnQueueFull is set to true.
	// Blocking option should be used carefully as it can severely affect the performance of an
	// application.
	BlockOnQueueFull bool

	// MaxPacketSize is the maximum number of packet size that will forcefully trigger a batch process.
	// The deault value of MaxPacketSize is 2M (in bytes) .
	MaxPacketSize int

	// EnabelSampler 启用采样器, 只上报采样的
	EnabelSampler bool
}

// WithMaxQueueSize WithMaxQueueSize
func WithMaxQueueSize(size int) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.MaxQueueSize = size
	}
}

// WithMaxExportBatchSize WithMaxExportBatchSize
func WithMaxExportBatchSize(size int) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.MaxExportBatchSize = size
	}
}

// WithBatchTimeout WithBatchTimeout
func WithBatchTimeout(delay time.Duration) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.BatchTimeout = delay
	}
}

// WithBlocking WithBlocking
func WithBlocking() BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.BlockOnQueueFull = true
	}
}

// WithMaxPacketSize WithMaxPacketSize
func WithMaxPacketSize(size int) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.MaxPacketSize = size
	}
}

// WithEnabelSampler 启用采样器
func WithEnabelSampler(enableSampler bool) BatchSyncerOption {
	return func(o *BatchSyncerOptions) {
		o.EnabelSampler = enableSampler
	}
}

// calcLogSize calculates the packet size of a ResourceLogs
func calcLogSize(rl *logsproto.ResourceLogs) int {
	return v1proto.Size(rl)
}
