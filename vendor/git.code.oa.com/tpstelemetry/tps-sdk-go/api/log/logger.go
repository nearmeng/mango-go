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

// Package log 天机阁log接口
package log

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
)

var logger Logger = NewNopLogger()

var _ Logger = (*NopLogger)(nil)

// NewNopLogger New 方法
func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

// NopLogger noop logger
type NopLogger struct{}

// With 空实现
func (n *NopLogger) With(ctx context.Context, values []attribute.KeyValue) context.Context {
	return ctx
}

// Log 空实现
func (n *NopLogger) Log(ctx context.Context, msg string, opts ...Option) {
}

// Logger 接口定义
type Logger interface {
	Log(context.Context, string, ...Option)
	With(context.Context, []attribute.KeyValue) context.Context
}

// GlobalLogger 获取全局 logger
func GlobalLogger() Logger {
	return logger
}

// SetGlobalLogger 注册全局 logger
func SetGlobalLogger(l Logger) {
	logger = l
}

type ctxMarker struct{}

var (
	ctxKey = &ctxMarker{}
)

// FromContext context 中提取 trace 属性信息
func FromContext(ctx context.Context) []attribute.KeyValue {
	l, ok := ctx.Value(ctxKey).([]attribute.KeyValue)

	if !ok || l == nil {
		return []attribute.KeyValue{}
	}

	return l
}

// ContextWith context 设置 trace 属性信息
func ContextWith(ctx context.Context, kvs []attribute.KeyValue) context.Context {
	labels := FromContext(ctx)
	labels = append(labels, kvs...)
	return context.WithValue(ctx, ctxKey, labels)
}
