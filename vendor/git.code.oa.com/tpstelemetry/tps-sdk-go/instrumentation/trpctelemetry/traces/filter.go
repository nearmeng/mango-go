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

// Package traces trpc traces 组件
package traces

import (
	"context"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/log"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/config"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/logs"

	tkafka "git.code.oa.com/trpc-go/trpc-database/kafka"
	thttp "git.code.oa.com/trpc-go/trpc-go/http"

	tpsapi "git.code.oa.com/tpstelemetry/tps-sdk-go/api"
	trpcsemconv "git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/semconv"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/metric"
	sdktrace "git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/trace"
)

type supplier struct {
	md           codec.MetaData
	serverHeader *thttp.Header
	keys         []string
}

// newSupplier
func newSupplier(md codec.MetaData, msg codec.Msg) *supplier {
	serverHeader := thttp.Head(msg.Context())
	return &supplier{
		md:           md,
		serverHeader: serverHeader,
		keys:         make([]string, 0),
	}
}

// Get 从业务字段中提取透传 trace
func (s supplier) Get(key string) string {
	value := s.md[key]
	if len(value) == 0 {
		if s.serverHeader != nil && s.serverHeader.Request != nil && s.serverHeader.Request.Header != nil {
			return s.serverHeader.Request.Header.Get(key)
		}
		return ""
	}
	return string(value)
}

// Set 将 trace 信息写入业务字段中
func (s supplier) Set(key string, value string) {
	s.md[key] = []byte(value)
	s.keys = append(s.keys, key)
}

// Keys trace 信息 key 列表
func (s supplier) Keys() []string {
	return s.keys
}

var (
	defaultTracer     trace.Tracer
	defaultTracerOnce sync.Once
)

func getDefaultTracer() trace.Tracer {
	defaultTracerOnce.Do(func() {
		defaultTracer = otel.Tracer("")
	})
	return defaultTracer
}

// FilterOptions
type FilterOptions struct {
	TraceLogMode     config.LogMode
	DisableTraceBody bool
}

// FilterOption filter option
type FilterOption func(*FilterOptions)

var defaultFilterOptions = FilterOptions{
	TraceLogMode:     config.LogModeOneLine,
	DisableTraceBody: false,
}

// ServerFilter tpstelemetry server filter in trpc
func ServerFilter(opts ...FilterOption) filter.Filter {
	opt := defaultFilterOptions
	for _, v := range opts {
		v(&opt)
	}
	return func(ctx context.Context, req, rsp interface{}, handle filter.HandleFunc) (err error) {
		start := time.Now()
		msg := trpc.Message(ctx)
		md := msg.ServerMetaData()
		if md == nil {
			md = codec.MetaData{}
		}
		// trace extract kafkaheader ---> md ---> ctx
		if kafkaMsg, ok := msg.ServerReqHead().(*sarama.ConsumerMessage); ok {
			traceParent, traceState, baggage := ExtractTraceContextFromKafkaHead(kafkaMsg)
			md[traceparentHeader] = traceParent
			md[tracestateHeader] = traceState
			md[baggageHeader] = baggage
		}

		ctx, span := startServerSpan(ctx, req, msg, md)
		defer span.End()

		flow := &logs.FlowLog{}
		flow.Source = logs.Service{
			Name:   msg.CallerServiceName(),
			Method: msg.CallerMethod(),
		}
		if msg.RemoteAddr() != nil {
			flow.Source.Address = msg.RemoteAddr().String()
		}
		flow.Target = logs.Service{
			Name:      msg.CalleeServiceName(),
			Method:    msg.CalleeMethod(),
			Namespace: msg.EnvName(),
		}
		if msg.LocalAddr() != nil {
			flow.Target.Address = msg.LocalAddr().String()
		}
		flow.Kind = logs.FlowKind(trace.SpanKindServer)
		isTraced := needToTraceBody(span, opt)
		if isTraced {
			reqStr := addEvent(ctx, req, semconv.RPCMessageTypeReceived)
			flow.Request.Body = reqStr
		}

		log.WithContextFields(ctx, "traceID", span.SpanContext().TraceID().String(),
			"spanID", span.SpanContext().SpanID().String(),
			"sampled", strconv.FormatBool(span.SpanContext().IsSampled()))
		err = handle(ctx, req, rsp)
		handleError(DefaultGetErrFunc(ctx, rsp, err), span, flow)
		span.SetAttributes(DefaultAttributesAfterServerHandle(ctx, rsp)...)
		if isTraced {
			rspStr := addEvent(ctx, rsp, semconv.RPCMessageTypeSent)
			flow.Response.Body = rspStr
		}
		flow.Cost = time.Since(start).String()
		doFlowLog(ctx, flow, opt)
		return err
	}
}

func startServerSpan(ctx context.Context,
	req interface{}, msg codec.Msg, md codec.MetaData) (context.Context, trace.Span) {
	ctx = otel.GetTextMapPropagator().Extract(ctx, newSupplier(md, msg))
	labelSet := baggage.Set(ctx)
	spanContext := trace.RemoteSpanContextFromContext(ctx)
	ctx = baggage.ContextWithValues(ctx, (&labelSet).ToSlice()...)

	var spanKind trace.SpanKind
	if _, ok := msg.ServerReqHead().(*sarama.ConsumerMessage); ok {
		spanKind = trace.SpanKindConsumer
	} else {
		spanKind = trace.SpanKindServer
	}

	return getDefaultTracer().Start(
		trace.ContextWithRemoteSpanContext(ctx, spanContext),
		msg.ServerRPCName(),
		trace.WithSpanKind(spanKind),
		trace.WithAttributes(peerInfo(msg.RemoteAddr())...),
		trace.WithAttributes(hostInfo(msg.LocalAddr())...),
		trace.WithAttributes(trpcsemconv.CallerServiceKey.String(msg.CallerServiceName())),
		// 只信任通过RegisterMethodMapping的pattern，避免大量脏数据写入拓扑存储中
		trace.WithAttributes(trpcsemconv.CallerMethodKey.String(metric.CleanRPCMethod(msg.CallerMethod()))),
		trace.WithAttributes(trpcsemconv.CalleeServiceKey.String(msg.CalleeServiceName())),
		// 只信任通过RegisterMethodMapping的pattern，避免大量脏数据写入拓扑存储中
		trace.WithAttributes(trpcsemconv.CalleeMethodKey.String(metric.CleanRPCMethod(msg.CalleeMethod()))),
		trace.WithAttributes(trpcsemconv.NamespaceKey.String(msg.Namespace()),
			trpcsemconv.EnvNameKey.String(msg.EnvName())),
		trace.WithAttributes(fromTRPCDyeingKey(msg.DyeingKey())...),
		trace.WithAttributes(forceSampleFromMetadata(msg.ServerMetaData())...),
		trace.WithAttributes(DefaultTraceAttributesFunc(ctx, req)...),
		trace.WithAttributes((&labelSet).ToSlice()...))
}

func needToTraceBody(span trace.Span, opt FilterOptions) bool {
	if opt.DisableTraceBody {
		return false
	}
	return span.SpanContext().IsSampled() || (opt.TraceLogMode != config.LogModeDisable)
}

func handleError(err error, span trace.Span, flow *logs.FlowLog) {
	if err == nil {
		span.SetStatus(codes.Ok, "")
		return
	}

	if e, ok := err.(*errs.Error); ok {
		span.SetAttributes(
			trpcsemconv.StatusCode.Int64(int64(e.Code)),
			trpcsemconv.StatusMsg.String(e.Msg),
			trpcsemconv.StatusType.Int(e.Type),
		)
		if e.Code != 0 {
			span.SetStatus(codes.Error, e.Msg)
		} else {
			span.SetStatus(codes.Ok, e.Msg)
		}
		flow.Status = logs.Status{
			Code:    e.Code,
			Message: e.Msg,
			Type:    toErrorType(e.Type),
		}
	} else {
		span.SetAttributes(
			trpcsemconv.StatusCode.Int64(errs.RetUnknown),
			trpcsemconv.StatusMsg.String(err.Error()),
		)
		span.SetStatus(codes.Error, err.Error())
		flow.Status = logs.Status{
			Code:    errs.RetUnknown,
			Message: err.Error(),
		}
	}
}

// ClientFilter tpstelemetry client filter in trpc
func ClientFilter(opts ...FilterOption) filter.Filter {
	opt := defaultFilterOptions
	for _, v := range opts {
		v(&opt)
	}
	return func(ctx context.Context, req interface{}, rsp interface{}, f filter.HandleFunc) (err error) {
		start := time.Now()
		msg := trpc.Message(ctx)
		md := msg.ClientMetaData()
		if md == nil {
			md = codec.MetaData{}
		}

		ctx, span := startClientSpan(ctx, req, msg)
		defer span.End()

		// 默认的trpc 注入
		supplierHandle := newSupplier(md, msg)
		otel.GetTextMapPropagator().Inject(ctx, supplierHandle)
		msg.WithClientMetaData(md)
		// kafka 注入, trace inject  ctx ---> md/httpHeader ---> kafkaHeader
		if req, ok := (msg.ClientReqHead()).(*tkafka.Request); ok {
			InjectTraceContextToKafkaHead(req, supplierHandle)
			msg.WithClientReqHead(req)
		}

		flow := &logs.FlowLog{
			Source: logs.Service{
				Name:      msg.CallerServiceName(),
				Method:    msg.CallerMethod(),
				Namespace: msg.EnvName(),
			},
			Target: logs.Service{
				Name:   msg.CalleeServiceName(),
				Method: msg.CalleeMethod(),
			},
		}
		flow.Kind = logs.FlowKind(trace.SpanKindClient)

		isTraced := needToTraceBody(span, opt)
		if isTraced {
			reqStr := addEvent(ctx, req, semconv.RPCMessageTypeSent)
			flow.Request.Body = reqStr
		}

		err = f(ctx, req, rsp)
		handleError(DefaultGetErrFunc(ctx, rsp, err), span, flow)
		span.SetAttributes(DefaultAttributesAfterClientHandle(ctx, rsp)...)
		span.SetAttributes(peerInfo(msg.RemoteAddr())...)
		span.SetAttributes(hostInfo(msg.LocalAddr())...)
		if isTraced {
			rspStr := addEvent(ctx, rsp, semconv.RPCMessageTypeReceived)
			flow.Response.Body = rspStr
		}
		flow.Cost = time.Since(start).String()
		if msg.RemoteAddr() != nil {
			flow.Target.Address = msg.RemoteAddr().String()
		}
		if msg.LocalAddr() != nil {
			flow.Source.Address = msg.LocalAddr().String()
		}
		doFlowLog(ctx, flow, opt)
		return err
	}
}

func startClientSpan(ctx context.Context, req interface{}, msg codec.Msg) (context.Context, trace.Span) {
	// 区分kafka还是rpc 请求
	var spanKind trace.SpanKind
	if _, ok := (msg.ClientReqHead()).(*tkafka.Request); ok {
		spanKind = trace.SpanKindProducer
	} else {
		// 默认 rpc client,后面可以扩展其他类型
		spanKind = trace.SpanKindClient
	}
	return getDefaultTracer().Start(ctx,
		msg.ClientRPCName(),
		trace.WithSpanKind(spanKind),
		trace.WithAttributes(trpcsemconv.CallerServiceKey.String(msg.CallerServiceName())),
		// 只信任通过RegisterMethodMapping的pattern，避免大量脏数据写入拓扑存储中
		trace.WithAttributes(trpcsemconv.CallerMethodKey.String(metric.CleanRPCMethod(msg.CallerMethod()))),
		trace.WithAttributes(trpcsemconv.CalleeServiceKey.String(msg.CalleeServiceName())),
		// 只信任通过RegisterMethodMapping的pattern，避免大量脏数据写入拓扑存储中
		trace.WithAttributes(trpcsemconv.CalleeMethodKey.String(metric.CleanRPCMethod(msg.CalleeMethod()))),
		trace.WithAttributes(trpcsemconv.NamespaceKey.String(msg.Namespace()),
			trpcsemconv.EnvNameKey.String(msg.EnvName())),
		trace.WithAttributes(fromTRPCDyeingKey(msg.DyeingKey())...),
		trace.WithAttributes(forceSampleFromMetadata(msg.ClientMetaData())...),
		trace.WithAttributes(DefaultTraceAttributesFunc(ctx, req)...))
}

func toErrorType(t int) string {
	switch t {
	case errs.ErrorTypeBusiness:
		return "business"
	case errs.ErrorTypeCalleeFramework:
		return "callee_framework"
	case errs.ErrorTypeFramework:
		return "framework"
	default:
		return ""
	}
}

const localhost = "127.0.0.1"

func peerInfo(addr net.Addr) []attribute.KeyValue {
	if addr == nil {
		return nil
	}
	host, port, err := net.SplitHostPort(addr.String())

	if err != nil {
		return []attribute.KeyValue{}
	}

	if host == "" {
		host = localhost
	}

	return []attribute.KeyValue{
		semconv.NetPeerIPKey.String(host),
		semconv.NetPeerPortKey.String(port),
	}
}

func hostInfo(addr net.Addr) []attribute.KeyValue {
	if addr == nil {
		return []attribute.KeyValue{
			semconv.NetHostNameKey.String(getHostname()),
		}
	}
	host, port, err := net.SplitHostPort(addr.String())

	if err != nil {
		return []attribute.KeyValue{
			semconv.NetHostNameKey.String(getHostname()),
		}
	}

	if host == "" {
		host = localhost
	}

	return []attribute.KeyValue{
		semconv.NetHostIPKey.String(host),
		semconv.NetHostPortKey.String(port),
		semconv.NetHostNameKey.String(getHostname()),
	}
}

var (
	hostname     string
	hostnameOnce sync.Once
)

// getHostname
func getHostname() string {
	hostnameOnce.Do(func() {
		hostname, _ = os.Hostname()
	})
	return hostname
}

// TraceAttributesFunc 从ctx和req中提取染色的trace attribute的hook函数
type TraceAttributesFunc func(ctx context.Context, req interface{}) []attribute.KeyValue

// DefaultTraceAttributesFunc 业务方在main函数中自定义的函数, 用于从ctx和req中提取染色的trace attribute
var DefaultTraceAttributesFunc TraceAttributesFunc = func(ctx context.Context, req interface{}) []attribute.KeyValue {
	return nil
}

// TraceEventMsgMarshaler trace event msg序列化方式
type TraceEventMsgMarshaler func(message interface{}) string

// DefaulTraceEventMsgMarshaler 默认序列化方式，业务方可以实现 TraceEventMsgMarshaler 类型来定制序列化
var DefaulTraceEventMsgMarshaler TraceEventMsgMarshaler = ProtoMessageToCustomJSONString

var (
	metadataKeyTPSForceSample = "tps-force-sample"
)

// GetErrFunc 根据ctx、rsp来重新设置err，以支持span Status以及flow Status正确设置的hook函数
type GetErrFunc func(ctx context.Context, rsp interface{}, err error) error

// DefaultGetErrFunc 业务方自定义的实现函数，根据ctx、rsp来重新设置err，以支持span Status以及flow Status的正确设置
var DefaultGetErrFunc GetErrFunc = func(ctx context.Context, rsp interface{}, err error) error {
	return err
}

// AttributesAfterHandle hook函数，filter内层handle执行后才能提取到的attributes
type AttributesAfterHandle func(ctx context.Context, rsp interface{}) []attribute.KeyValue

// DefaultAttributesAfterServerHandle serverFilter内层handle执行后才能提取的attributes，业务方自定义
var DefaultAttributesAfterServerHandle AttributesAfterHandle = func(ctx context.Context,
	rsp interface{}) []attribute.KeyValue {
	return nil
}

// DefaultAttributesAfterClientHandle clientFilter内层handle执行后才能提取的attributes，业务方自定义
var DefaultAttributesAfterClientHandle AttributesAfterHandle = func(ctx context.Context,
	rsp interface{}) []attribute.KeyValue {
	return nil
}

// forceSampleFromMetadata 用户可以手动请求时在metadata中附加一个tps-force-sample, 则此请求必采样
func forceSampleFromMetadata(metadata codec.MetaData) []attribute.KeyValue {
	v := metadata[metadataKeyTPSForceSample]
	if len(v) == 0 {
		return nil
	}
	return []attribute.KeyValue{sdktrace.TPSForceSampler.String(string(v))}
}

// fromTRPCDyeingKey 从tRPC的染色key中提取attribute
func fromTRPCDyeingKey(dyeingKey string) []attribute.KeyValue {
	if dyeingKey == "" {
		return nil
	}
	return []attribute.KeyValue{tpsapi.TpsDyeingKey.String(dyeingKey)}
}
