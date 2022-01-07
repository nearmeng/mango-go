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
	"time"

	commonproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/common/v1"
	logsproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/logs/v1"
	resourceproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/resource/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/api/log"
)

var _ log.Logger = (*Logger)(nil)

// NewLogger Logger 工厂方法
func NewLogger(opts ...LoggerOption) *Logger {
	options := &LoggerOptions{}
	for _, o := range opts {
		o(options)
	}

	return &Logger{
		opts: options,
	}
}

// LoggerOptions logger options detail
type LoggerOptions struct {
	// Resource contains attributes representing an entity that produces telemetry.
	Resource *resource.Resource

	// Processor export logs
	Processor *BatchProcessor

	// LevelEnabled enabled level
	LevelEnabled log.Level

	// EnableSampler enable sampler.
	EnableSampler bool
}

// LoggerOption logger option func
type LoggerOption func(*LoggerOptions)

// WithLevelEnable enable level
func WithLevelEnable(level log.Level) LoggerOption {
	return func(options *LoggerOptions) {
		options.LevelEnabled = level
	}
}

// WithEnableSampler enable sample along with trace.
func WithEnableSampler(enableSampler bool) LoggerOption {
	return func(options *LoggerOptions) {
		options.EnableSampler = enableSampler
	}
}

// WithResource setting resource info
func WithResource(rs *resource.Resource) LoggerOption {
	return func(options *LoggerOptions) {
		options.Resource = rs
	}
}

// WithBatcher 指定批处理器
func WithBatcher(batcher *BatchProcessor) LoggerOption {
	return func(options *LoggerOptions) {
		options.Processor = batcher
	}
}

// Logger logger impl
type Logger struct {
	opts *LoggerOptions
}

// Shutdown 关闭方法
func (l *Logger) Shutdown(ctx context.Context) error {
	return l.opts.Processor.Shutdown(ctx)
}

// With 指定附加属性
func (l *Logger) With(ctx context.Context, values []attribute.KeyValue) context.Context {
	return log.ContextWith(ctx, values)
}

// Log record a log
func (l *Logger) Log(ctx context.Context, msg string, opts ...log.Option) {
	cfg := &log.Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	sampled := false
	// 开启日志采样.
	if l.opts.EnableSampler && trace.SpanFromContext(ctx).SpanContext().IsSampled() &&
		cfg.Level >= l.opts.LevelEnabled {
		sampled = true
	}
	// 未开启采样.
	// 日志级别比设置开启的日志级别小，则必定采样命中上报
	if !l.opts.EnableSampler && cfg.Level >= l.opts.LevelEnabled {
		sampled = true
	}

	l.log(ctx, msg, cfg, sampled)
}

func toSeverityNumber(level log.Level) logsproto.SeverityNumber {
	var number logsproto.SeverityNumber
	switch level {
	case log.TraceLevel:
		number = logsproto.SeverityNumber_TRACE
	case log.DebugLevel:
		number = logsproto.SeverityNumber_DEBUG
	case log.InfoLevel:
		number = logsproto.SeverityNumber_INFO
	case log.WarnLevel:
		number = logsproto.SeverityNumber_WARN
	case log.ErrorLevel:
		number = logsproto.SeverityNumber_ERROR
	case log.FatalLevel:
		number = logsproto.SeverityNumber_FATAL
	}

	return number
}

func (l *Logger) log(ctx context.Context, msg string, cfg *log.Config, sampled bool) {
	span := trace.SpanFromContext(ctx)
	//  如果逻辑到此依旧没有命中，则直接返回不进行上报
	if !sampled {
		return
	}
	record := &logsproto.LogRecord{
		TimeUnixNano:   uint64(time.Now().UnixNano()),
		SeverityText:   string(cfg.Level),
		SeverityNumber: toSeverityNumber(cfg.Level),
	}
	record.Body = &commonproto.AnyValue{
		Value: &commonproto.AnyValue_StringValue{StringValue: msg},
	}
	record.Flags = uint32(span.SpanContext().TraceFlags())
	kvs := log.FromContext(ctx)
	cfg.Fields = append(cfg.Fields, kvs...)
	if span.SpanContext().IsSampled() {
		traceID := span.SpanContext().TraceID()
		spanID := span.SpanContext().SpanID()
		if span != nil {
			if span.SpanContext().HasSpanID() {
				record.SpanId = spanID[:]
			}
			if span.SpanContext().HasTraceID() {
				record.TraceId = traceID[:]
			}
		}
	}
	for _, field := range cfg.Fields {
		record.Attributes = append(record.Attributes, toAttribute(field))
	}
	logs := &logsproto.ResourceLogs{
		Resource: Resource(l.opts.Resource),
		InstrumentationLibraryLogs: []*logsproto.InstrumentationLibraryLogs{
			{
				Logs: []*logsproto.LogRecord{record},
			},
		},
	}
	l.opts.Processor.Enqueue(logs)
}

func toAttribute(v attribute.KeyValue) *commonproto.KeyValue {
	switch v.Value.Type() {
	case attribute.BOOL:
		return &commonproto.KeyValue{
			Key: string(v.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_BoolValue{BoolValue: v.Value.AsBool()},
			},
		}
	case attribute.INT64:
		return &commonproto.KeyValue{
			Key: string(v.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_IntValue{IntValue: v.Value.AsInt64()},
			},
		}
	case attribute.FLOAT64:
		return &commonproto.KeyValue{
			Key: string(v.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_DoubleValue{DoubleValue: v.Value.AsFloat64()},
			},
		}
	case attribute.STRING:
		return &commonproto.KeyValue{
			Key: string(v.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_StringValue{StringValue: v.Value.AsString()},
			},
		}
	default:
		return nil
	}
}

// Resource transforms a Resource into an OTLP Resource.
func Resource(r *resource.Resource) *resourceproto.Resource {
	if r == nil {
		return nil
	}
	return &resourceproto.Resource{Attributes: ResourceAttributes(r)}
}

// ResourceAttributes transforms a Resource into a slice of OTLP attribute key-values.
func ResourceAttributes(resource *resource.Resource) []*commonproto.KeyValue {
	if resource.Len() == 0 {
		return nil
	}

	out := make([]*commonproto.KeyValue, 0, resource.Len())
	for iter := resource.Iter(); iter.Next(); {
		out = append(out, toAttribute(iter.Attribute()))
	}

	return out
}
