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

// Package trace trace 组件
package trace

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/pkg/metrics"
)

var _ sdktrace.SpanProcessor = (*DeferredSampleProcessor)(nil)

// DeferredSampler 延迟采样, 处理span.End之后的过滤条件 返回true则保留, 返回false则drop
type DeferredSampler func(sdktrace.ReadOnlySpan) bool

// DeferredSampleConfig 延迟采样配置
type DeferredSampleConfig struct {
	Enabled            bool          // 是否启用, 不启用则不会过滤
	SampleError        bool          // 采样出错的
	SampleSlowDuration time.Duration // 采样慢操作的
}

// NewDeferredSampler 根据选项创建一个
func NewDeferredSampler(cfg DeferredSampleConfig) DeferredSampler {
	sampledCounter := metrics.DeferredProcessCounter.WithLabelValues("sampled", "traces")
	errorCounter := metrics.DeferredProcessCounter.WithLabelValues("deferred_error", "traces")
	slowCounter := metrics.DeferredProcessCounter.WithLabelValues("deferred_slow", "traces")
	unsampledCounter := metrics.DeferredProcessCounter.WithLabelValues("unsampled", "traces")
	return func(s sdktrace.ReadOnlySpan) bool {
		// 已经采样的
		if s.SpanContext().IsSampled() {
			sampledCounter.Inc()
			return true
		}
		if cfg.Enabled && cfg.SampleError && s.StatusCode() != codes.Ok {
			// 出错的
			errorCounter.Inc()
			return true
		}
		if cfg.Enabled && cfg.SampleSlowDuration != 0 && s.EndTime().Sub(s.StartTime()) >= cfg.SampleSlowDuration {
			// 高耗时的
			slowCounter.Inc()
			return true
		}
		unsampledCounter.Inc()
		return false
	}
}

// DeferredSampleProcessor 延迟采样processor, 处理span.End之后的过滤条件
type DeferredSampleProcessor struct {
	next            sdktrace.SpanProcessor
	deferredSampler DeferredSampler
}

// NewDeferredSampleProcessor 创建一个延迟采样processor
func NewDeferredSampleProcessor(next sdktrace.SpanProcessor,
	sampleFunc func(sdktrace.ReadOnlySpan) bool) *DeferredSampleProcessor {
	return &DeferredSampleProcessor{
		next:            next,
		deferredSampler: sampleFunc,
	}
}

// OnStart is called when a span is started. It is called synchronously
// and should not block.
func (p *DeferredSampleProcessor) OnStart(parent context.Context, s sdktrace.ReadWriteSpan) {
	p.next.OnStart(parent, s)
}

// OnEnd is called when span is finished. It is called synchronously and
// hence not block.
func (p *DeferredSampleProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	if p.deferredSampler == nil {
		// keep
		p.next.OnEnd(s)
		return
	}
	if p.deferredSampler(s) {
		// keep
		p.next.OnEnd(s)
		return
	}
	// drop
}

// Shutdown is called when the SDK shuts down. Any cleanup or release of
// resources held by the processor should be done in this call.
//
// Calls to OnStart, OnEnd, or ForceFlush after this has been called
// should be ignored.
//
// All timeouts and cancellations contained in ctx must be honored, this
// should not block indefinitely.
func (p *DeferredSampleProcessor) Shutdown(ctx context.Context) error {
	return p.next.Shutdown(ctx)
}

// ForceFlush exports all ended spans to the configured Exporter that have not yet
// been exported.  It should only be called when absolutely necessary, such as when
// using a FaaS provider that may suspend the process after an invocation, but before
// the Processor can export the completed spans.
func (p *DeferredSampleProcessor) ForceFlush(ctx context.Context) error {
	return p.next.ForceFlush(ctx)
}
