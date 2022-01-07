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

// Package logs 日志组件
package logs

import (
	"context"
	"fmt"
	"strings"

	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/log"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// FlowLog log model for rpc
type FlowLog struct {
	Kind     FlowKind `json:"kind,omitempty"`
	Source   Service  `json:"source,omitempty"`
	Target   Service  `json:"target,omitempty"`
	Request  Request  `json:"request,omitempty"`
	Response Response `json:"response,omitempty"`
	Cost     string   `json:"cost,omitempty"`
	Status   Status   `json:"status,omitempty"`
}

// String ...
func (f FlowLog) String() string {
	return f.OneLineString()
}

// MultilineString ...
func (f FlowLog) MultilineString() string {
	var sb strings.Builder
	switch trace.SpanKind(f.Kind) {
	case trace.SpanKindServer:
		sb.WriteString(fmt.Sprintf("[FLOW(%s)] %s\n", f.Kind.String(), f.Target.String()))
		sb.WriteString(fmt.Sprintf("RecvFrom: %s\n", f.Source.String()))
	case trace.SpanKindClient:
		sb.WriteString(fmt.Sprintf("[FLOW(%s)] %s\n", f.Kind.String(), f.Source.String()))
		sb.WriteString(fmt.Sprintf("SentTo: %s\n", f.Target.String()))
	}
	sb.WriteString(fmt.Sprintf("Status: %s\n", f.Status))
	sb.WriteString(fmt.Sprintf("Cost: %s\n", f.Cost))
	sb.WriteString(fmt.Sprintf("Request.Body: %s\n", f.Request.Body))
	sb.WriteString(fmt.Sprintf("Response.Body: %s\n", f.Response.Body))
	return sb.String()
}

// OneLineString ...
func (f FlowLog) OneLineString() string {
	var sb strings.Builder
	switch trace.SpanKind(f.Kind) {
	case trace.SpanKindServer:
		sb.WriteString(fmt.Sprintf("[FLOW(%s)] %s ", f.Kind.String(), f.Target.String()))
		sb.WriteString(fmt.Sprintf(" RecvFrom: %s ", f.Source.String()))
	case trace.SpanKindClient:
		sb.WriteString(fmt.Sprintf(" [FLOW(%s)] %s ", f.Kind.String(), f.Source.String()))
		sb.WriteString(fmt.Sprintf(" SentTo: %s ", f.Target.String()))
	}
	sb.WriteString(fmt.Sprintf(" Status: %s ", f.Status))
	sb.WriteString(fmt.Sprintf(" Cost: %s ", f.Cost))
	sb.WriteString(fmt.Sprintf(" Request.Body: %s ", f.Request.Body))
	sb.WriteString(fmt.Sprintf(" Response.Body: %s", f.Response.Body))
	return sb.String()
}

type FlowKind trace.SpanKind

// MarshalJSON SpanKind 序列化方式
func (k FlowKind) MarshalJSON() ([]byte, error) {
	switch k {
	case FlowKindServer:
		return []byte("\"server\""), nil
	case FlowKindClient:
		return []byte("\"client\""), nil
	default:
		return []byte("\"internal\""), nil
	}
}

const (
	FlowKindServer FlowKind = FlowKind(trace.SpanKindServer)
	FlowKindClient FlowKind = FlowKind(trace.SpanKindClient)
)

// MarshalJSON SpanKind String 化
func (k FlowKind) String() string {
	switch trace.SpanKind(k) {
	case trace.SpanKindServer:
		return "SERVER"
	case trace.SpanKindClient:
		return "CLIENT"
	default:
		return "INTERNAL"
	}
}

// Status rpc 调用状态信息描述
type Status struct {
	Code    int32  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
}

// Service service 信息描述
type Service struct {
	Name      string `json:"service,omitempty"`
	Method    string `json:"method,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Address   string `json:"address,omitempty"`
}

// String Status string 化
func (s Status) String() string {
	return fmt.Sprintf("%d %s(%s)", s.Code, s.Type, s.Message)
}

// String Service string 化
func (s Service) String() string {
	return fmt.Sprintf("%s/%s@%s(%s)", s.Name, s.Method, s.Namespace, s.Address)
}

// Request 请求体描述
type Request struct {
	Head string `json:"head,omitempty"`
	Body string `json:"body,omitempty"`
}

// Response 回包体描述
type Response struct {
	Head string `json:"head,omitempty"`
	Body string `json:"body,omitempty"`
}

func spanLogf(ctx context.Context, level log.Level, format string, v []interface{}) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	var msg string
	if format == "" {
		msg = fmt.Sprint(v...)
	} else {
		msg = fmt.Sprintf(format, v...)
	}
	span.AddEvent("", trace.WithAttributes(attribute.String("msg", msg), attribute.String("level", level.String())))
}

// Debug Debug 日志级别 log 打印 helper
func Debug(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelDebug, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Debug(args...)
			return
		}
		l.GetLogger().Debug(args...)
	case log.Logger:
		l.Debug(args...)
	default:
		log.DefaultLogger.Debug(args...)
	}
}

// Debugf Debug 日志级别 log 打印 helper
func Debugf(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelDebug, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Debugf(format, args...)
			return
		}
		l.GetLogger().Debugf(format, args...)
	case log.Logger:
		l.Debugf(format, args...)
	default:
		log.DefaultLogger.Debugf(format, args...)
	}
}

// Info Info 日志级别 log 打印 helper
func Info(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelInfo, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Info(args...)
			return
		}
		l.GetLogger().Info(args...)
	case log.Logger:
		l.Info(args...)
	default:
		log.DefaultLogger.Info(args...)
	}
}

// Infof Info 日志级别 log 打印 helper
func Infof(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelInfo, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Infof(format, args...)
			return
		}
		l.GetLogger().Infof(format, args...)
	case log.Logger:
		l.Infof(format, args...)
	default:
		log.DefaultLogger.Infof(format, args...)
	}
}

// Warn Warn 日志级别 log 打印 helper
func Warn(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelWarn, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Warn(args...)
			return
		}
		l.GetLogger().Warn(args...)
	case log.Logger:
		l.Warn(args...)
	default:
		log.DefaultLogger.Warn(args...)
	}
}

// Warnf Warn 日志级别 log 打印 helper
func Warnf(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelWarn, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Warnf(format, args...)
			return
		}
		l.GetLogger().Warnf(format, args...)
	case log.Logger:
		l.Warnf(format, args...)
	default:
		log.DefaultLogger.Warnf(format, args...)
	}
}

// Error Error 日志级别 log 打印 helper
func Error(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelError, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Error(args...)
			return
		}
		l.GetLogger().Error(args...)
	case log.Logger:
		l.Error(args...)
	default:
		log.DefaultLogger.Error(args...)
	}
}

// Errorf Error 日志级别 log 打印 helper
func Errorf(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelError, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Errorf(format, args...)
			return
		}
		l.GetLogger().Errorf(format, args...)
	case log.Logger:
		l.Errorf(format, args...)
	default:
		log.DefaultLogger.Errorf(format, args...)
	}
}

// Fatal Fatal 日志级别 log 打印 helper
func Fatal(ctx context.Context, args ...interface{}) {
	spanLogf(ctx, log.LevelFatal, "", args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Fatal(args...)
			return
		}
		l.GetLogger().Fatal(args...)
	case log.Logger:
		l.Fatal(args...)
	default:
		log.DefaultLogger.Fatal(args...)
	}
}

// Fatalf Fatal 日志级别 log 打印 helper
func Fatalf(ctx context.Context, format string, args ...interface{}) {
	spanLogf(ctx, log.LevelFatal, format, args)

	switch l := codec.Message(ctx).Logger().(type) {
	case *log.ZapLogWrapper:
		if l == nil {
			log.DefaultLogger.Fatalf(format, args...)
			return
		}
		l.GetLogger().Fatalf(format, args...)
	case log.Logger:
		l.Fatalf(format, args...)
	default:
		log.DefaultLogger.Fatalf(format, args...)
	}
}
