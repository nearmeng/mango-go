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

// Package tpstelemetry  tpstelemetry
package tpstelemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/exporters/stdout"
	"go.opentelemetry.io/otel/propagation"
	export "go.opentelemetry.io/otel/sdk/export/trace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
	apitrace "go.opentelemetry.io/otel/trace"

	"google.golang.org/grpc"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/api"
	apilog "git.code.oa.com/tpstelemetry/tps-sdk-go/api/log"
	tpsotlp "git.code.oa.com/tpstelemetry/tps-sdk-go/exporter/otlp"
	sdklog "git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/log"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/trace"

	_ "google.golang.org/grpc/encoding/gzip" // open gzip
)

var globalTracer = apitrace.NewNoopTracerProvider().Tracer("")

var (
	DefaultTenantID     = "default"
	DefaultExporterAddr = "localhost:12520"
	DefaultLogLevel     = apilog.InfoLevel
	MaxSendMessageSize  = 4194304
)

// GlobalTracer global tracer
func GlobalTracer() apitrace.Tracer {
	return globalTracer
}

// Start 启动方法
func Start(ctx context.Context, spanName string, opts ...apitrace.SpanOption) (context.Context, apitrace.Span) {
	return globalTracer.Start(ctx, spanName, opts...)
}

// WithSpan 设置 span 属性
func WithSpan(ctx context.Context, spanName string, fn func(ctx context.Context) error,
	opts ...apitrace.SpanOption) error {
	ctx, sp := globalTracer.Start(ctx, spanName, opts...)
	defer sp.End()
	return fn(ctx)
}

// SyncSetup 天机阁初始化(同步上报接口)，常用于非常驻服务的工具中，工具会快速结束，为保证工具结束前遥测数据正常上报完成，需要使用此同步接口
func SyncSetup(addr string, tenantID string, sampler sdktrace.Sampler) error {
	return setup(addr, WithTenantID(tenantID), WithSampler(sampler), WithAsync(false))
}

// AsyncSetup 天机阁初始化(异步上报接口)，常用于常驻的服务中。通过异步累计数据进行批量上报，提高性能。
func AsyncSetup(addr string, tenantID string, sampler sdktrace.Sampler) error {
	return setup(addr, WithTenantID(tenantID), WithSampler(sampler), WithAsync(true))
}

// Setup 初始化
func Setup(addr string, opts ...SetupOption) error {
	return setup(addr, opts...)
}

func setup(addr string, options ...SetupOption) error {
	o := defaultSetupOptions()
	for _, opt := range options {
		opt(o)
	}

	otlpOpts := []otlpgrpc.Option{
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(addr),
		otlpgrpc.WithCompressor("gzip"),
		otlpgrpc.WithHeaders(map[string]string{api.TenantHeaderKey: o.tenantID}),
		otlpgrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(MaxSendMessageSize))),
	}

	if len(o.grpcDialOptions) > 0 {
		otlpOpts = append(otlpOpts, otlpgrpc.WithDialOption(o.grpcDialOptions...))
	}

	var exp export.SpanExporter
	var err error

	if !o.stdOut {
		driver := otlpgrpc.NewDriver(otlpOpts...)
		exp, err = otlp.NewExporter(context.Background(), driver)
		if err != nil {
			return err
		}
	} else {
		exp, err = stdout.NewExporter()
		if err != nil {
			return err
		}
	}

	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithSampler(o.sampler))
	if o.async {
		opts = append(opts, sdktrace.WithSpanProcessor(
			trace.NewDeferredSampleProcessor(
				trace.NewBatchSpanProcessor(exp), o.deferredSampler)))
	} else {
		opts = append(opts, sdktrace.WithSpanProcessor(
			trace.NewDeferredSampleProcessor(
				sdktrace.NewSimpleSpanProcessor(exp), o.deferredSampler)))
	}
	kvs := []attribute.KeyValue{
		api.TpsTenantIDKey.String(o.tenantID),
		api.TpsOwnerKey.String(o.ServerOwner),
		api.TpsCmdbIDKey.String(o.CmdbID),
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKNameKey.String(api.TpsTelemetryName),
	}
	kvs = append(kvs, o.additionalLabels...)
	if o.serviceName != "" {
		kvs = append(kvs, semconv.ServiceNameKey.String(o.serviceName))
	}
	if o.serviceNamespace != "" {
		kvs = append(kvs, semconv.ServiceNamespaceKey.String(o.serviceNamespace))
	}

	if o.logEnabled {
		if err = setupLog(addr, o, kvs); err != nil {
			return err
		}
	}

	res := resource.NewWithAttributes(kvs...)
	opts = append(opts, sdktrace.WithResource(res))

	traceProvider := sdktrace.NewTracerProvider(opts...)

	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	globalTracer = otel.Tracer("")

	return nil
}

func setupLog(addr string, o *setupOptions, kvs []attribute.KeyValue) error {
	exp, err := tpsotlp.NewExporter(
		tpsotlp.WithInsecure(),
		tpsotlp.WithAddress(addr),
		tpsotlp.WithTenantID(o.tenantID),
		tpsotlp.WithCompressor("gzip"),
		tpsotlp.WithHeaders(map[string]string{api.TenantHeaderKey: o.tenantID}),
	)
	if err != nil {
		return err
	}
	logger := sdklog.NewLogger(
		sdklog.WithResource(resource.NewWithAttributes(kvs...)),
		sdklog.WithBatcher(sdklog.NewBatchProcessor(exp)),
		sdklog.WithLevelEnable(o.enabledLogLevel),
	)
	apilog.SetGlobalLogger(logger)
	return nil
}

type setupOptions struct {
	tenantID         string
	sampler          sdktrace.Sampler
	async            bool
	serviceName      string
	serviceNamespace string
	grpcDialOptions  []grpc.DialOption
	resourceLabels   *resource.Resource
	logEnabled       bool
	enabledLogLevel  apilog.Level
	ServerOwner      string
	CmdbID           string
	additionalLabels []attribute.KeyValue
	deferredSampler  trace.DeferredSampler
	stdOut           bool // span不会上报到远端collector，只会在本地日志打印。用于本地sdk体验
}

func defaultSetupOptions() *setupOptions {
	return &setupOptions{
		tenantID:        DefaultTenantID,
		sampler:         sdktrace.AlwaysSample(),
		async:           true,
		logEnabled:      false,
		enabledLogLevel: apilog.InfoLevel,
		stdOut:          false,
	}
}

type SetupOption func(*setupOptions)

// WithLogEnabled 是否开启log
func WithLogEnabled(enabled bool) SetupOption {
	return func(options *setupOptions) {
		options.logEnabled = enabled
	}
}

// WithLevelEnable 是否开启日志等级
func WithLevelEnable(level apilog.Level) SetupOption {
	return func(options *setupOptions) {
		options.enabledLogLevel = level
	}
}

// WithCmdbID 设置cmdb信息
func WithCmdbID(cmdbID string) SetupOption {
	return func(options *setupOptions) {
		options.CmdbID = cmdbID
	}
}

// WithServerOwner 设置服务owner
func WithServerOwner(owner string) SetupOption {
	return func(options *setupOptions) {
		options.ServerOwner = owner
	}
}

// WithTenantID 这只服务owner
func WithTenantID(tenantID string) SetupOption {
	return func(options *setupOptions) {
		options.tenantID = tenantID
	}
}

// WithResource WithResource
func WithResource(rs *resource.Resource) SetupOption {
	return func(options *setupOptions) {
		options.resourceLabels = rs
	}
}

// WithSampler 设置采样器
func WithSampler(sampler sdktrace.Sampler) SetupOption {
	return func(options *setupOptions) {
		options.sampler = sampler
	}
}

// WithAsync 设置异步
func WithAsync(async bool) SetupOption {
	return func(options *setupOptions) {
		options.async = async
	}
}

// WithServiceName 设置服务名
func WithServiceName(serviceName string) SetupOption {
	return func(options *setupOptions) {
		options.serviceName = serviceName
	}
}

// WithGRPCDialOption 设置 grpc 参数
func WithGRPCDialOption(opts ...grpc.DialOption) SetupOption {
	return func(cfg *setupOptions) {
		cfg.grpcDialOptions = opts
	}
}

// WithLabels 设置自定义tags
func WithLabels(opts ...attribute.KeyValue) SetupOption {
	return func(cfg *setupOptions) {
		cfg.additionalLabels = opts
	}
}

// WithServiceNamespace 设置namespace
func WithServiceNamespace(opts string) SetupOption {
	return func(cfg *setupOptions) {
		cfg.serviceNamespace = opts
	}
}

// WithDeferredSampler 传入延迟采样过滤函数
func WithDeferredSampler(deferredSampler trace.DeferredSampler) SetupOption {
	return func(cfg *setupOptions) {
		cfg.deferredSampler = deferredSampler
	}
}

// WithStdOutDebug 设置stdout
func WithStdOutDebug(stdOut bool) SetupOption {
	return func(cfg *setupOptions) {
		cfg.stdOut = true
	}
}

// Shutdown 进程结束前上传所有未上传数据
func Shutdown(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
	}
	if logger, ok := apilog.GlobalLogger().(*sdklog.Logger); ok {
		if err := logger.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}
