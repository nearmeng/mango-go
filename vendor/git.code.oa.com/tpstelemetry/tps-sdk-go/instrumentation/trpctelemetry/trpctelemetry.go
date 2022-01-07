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

// Package trpctelemetry trpc instrumentation
package trpctelemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"

	v1proto "github.com/golang/protobuf/proto"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/plugin"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/config"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/env"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/logs"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/traces"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/metric"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/remote"

	tpstelemetry "git.code.oa.com/tpstelemetry/tps-sdk-go"
	tpsprometheus "git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/metrics/prometheus"
	tpstrace "git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/trace"
)

const (
	pluginName = "tpstelemetry"
	pluginType = "telemetry"
)

func init() {
	Register()
	tpsprometheus.MonitorTRPCSDKMeta()
}

var _ plugin.Factory = (*factory)(nil)

type factory struct {
}

// Type Type方法
func (f factory) Type() string {
	return pluginType
}

// packetSizeMetric span的包大小监控
func packetSizeMetric() func(ctx context.Context, method string,
	req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		switch req := req.(type) {
		case proto.Message:
			tpsprometheus.ObserveExportSpansBytes(proto.Size(req))
		case v1proto.Message:
			tpsprometheus.ObserveExportSpansBytes(v1proto.Size(req))
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// DefaultRecoveryHandler 默认的recovery处理函数, 导出变量使得使用方可以自定义panic处理函数
var DefaultRecoveryHandler = func(ctx context.Context, panicErr interface{}) error {
	return fmt.Errorf("panic:%v", panicErr)
}

func recovery() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				buf := make([]byte, 2048)
				buf = buf[:runtime.Stack(buf, false)]
				// 如果是req序列化造成的panic, 不能再用%v %+v来打印, 否则可能会再次panic
				reqData, _ := json.Marshal(req)
				log.Printf("tpstelemetry: otel export panic:%v, req:%s %#v, stack:%s",
					rec, reqData, req, buf)
				tpsprometheus.IncrSDKPanicTotal()
				err = DefaultRecoveryHandler(ctx, rec)
			}
		}()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// Setup Setup 方法
func (f factory) Setup(name string, configDec plugin.Decoder) error {
	serverInfo, err := env.GetServerInfo()
	if err != nil {
		serverInfo = &env.ServerInfo{}
	}
	cfg := &config.Config{}
	*cfg = config.DefaultConfig()
	if err = configDec.Decode(cfg); err != nil {
		return err
	}

	cmdbID := serverInfo.CmdbID
	// 服务owner优先从配置里拿，如果没有，再用123接口查到的结果
	serverOwner := cfg.Metrics.ServerOwner
	if serverOwner == "" {
		serverOwner = serverInfo.Owner
	}

	sampler := tpstrace.NewSampler(cfg.Sampler.SamplerServerAddr, cfg.Sampler.SyncInterval, cfg.TenantID,
		cfg.Sampler.Fraction, func(opt *tpstrace.SamplerOptions) {
			if cfg.Traces.EnableDeferredSample {
				opt.DefaultSamplingDecision = sdktrace.RecordOnly
			}
		})
	DeferredSampler := tpstrace.NewDeferredSampler(tpstrace.DeferredSampleConfig{
		Enabled:            cfg.Traces.EnableDeferredSample,
		SampleError:        cfg.Traces.DeferredSampleError,
		SampleSlowDuration: cfg.Traces.DeferredSampleSlowDuration,
	})
	configurator := remote.NewRemoteConfigurator(cfg.Sampler.SamplerServerAddr, 0,
		cfg.TenantID, trpc.GlobalConfig().Server.App, trpc.GlobalConfig().Server.Server,
	)
	var serviceName string
	if len(trpc.GlobalConfig().Server.Service) > 0 {
		serviceName = trpc.GlobalConfig().Server.Service[0].Name
	}

	err = tpstelemetry.Setup(cfg.Addr,
		tpstelemetry.WithTenantID(cfg.TenantID),
		tpstelemetry.WithSampler(sampler),
		tpstelemetry.WithDeferredSampler(DeferredSampler),
		tpstelemetry.WithServiceName(serviceName),
		tpstelemetry.WithAsync(true),
		tpstelemetry.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(
			recovery(),
			grpc_prometheus.UnaryClientInterceptor,
			packetSizeMetric())),
		tpstelemetry.WithServerOwner(serverOwner),
		tpstelemetry.WithCmdbID(cmdbID))
	if err != nil {
		return err
	}
	tpsprometheus.Setup(cfg.TenantID, cfg.Metrics.RegistryEndpoints,
		metric.WithServerOwner(serverOwner),
		metric.WithCmdbID(cmdbID),
		metric.WithCodes(convertCodes(cfg.Metrics.CodeTypeMapping, cfg.Metrics.Codes)),
		metric.WithConfigurator(configurator),
	)
	filterOpts := func(o *traces.FilterOptions) {
		o.TraceLogMode = cfg.Logs.TraceLogMode
		o.DisableTraceBody = cfg.Traces.DisableTraceBody
	}
	logFilterOpts := func(o *logs.FilterOptions) {
		o.DisableRecovery = cfg.Logs.DisableRecovery
	}

	// override register filter with config options
	filter.Register(pluginName,
		filter.Chain{traces.ServerFilter(filterOpts), tpsprometheus.ServerFilter(),
			logs.LogRecoveryFilter(logFilterOpts)}.Handle,
		filter.Chain{traces.ClientFilter(filterOpts), tpsprometheus.ClientFilter()}.Handle,
	)
	// 指针替换以便之前使用server.WithFilter的也能生效
	sf := &ServerFilter
	*sf = filter.Chain{traces.ServerFilter(filterOpts), tpsprometheus.ServerFilter(),
		logs.LogRecoveryFilter(logFilterOpts)}.Handle
	cf := &ClientFilter
	*cf = filter.Chain{traces.ClientFilter(filterOpts), tpsprometheus.ClientFilter()}.Handle
	return nil
}

// convertCodes 合并trpc默认的错误码
func convertCodes(
	codeTypeMapping map[string]*metric.CodeTypeMappingDescription,
	codes []*metric.Code) []*metric.Code {
	defaultCodeTypeMapping := map[string]*metric.CodeTypeMappingDescription{
		"0":   metric.NewCodeTypeMappingDescription(metric.CodeTypeSuccess, "code=0"),
		"":    metric.NewCodeTypeMappingDescription(metric.CodeTypeSuccess, "code="),
		"101": metric.NewCodeTypeMappingDescription(metric.CodeTypeTimeout, "client timeout"),
		"21":  metric.NewCodeTypeMappingDescription(metric.CodeTypeTimeout, "server timeout"),
	}
	merged := make(map[string]*metric.CodeTypeMappingDescription, len(codeTypeMapping)+len(defaultCodeTypeMapping))
	for k, v := range codeTypeMapping {
		merged[k] = v
	}
	for code, v := range defaultCodeTypeMapping {
		if _, ok := merged[code]; !ok {
			merged[code] = v
		}
	}
	for k, v := range merged {
		codes = append(codes, metric.NewCode(k, metric.CodeType(v.CodeType), v.Description))
	}
	return codes
}

// Register 注册插件
func Register() {
	filter.Register(pluginName, ServerFilter, ClientFilter)
	plugin.Register(pluginName, &factory{})
}

var ServerFilter = filter.Chain{traces.ServerFilter(), tpsprometheus.ServerFilter(),
	logs.LogRecoveryFilter()}.Handle
var ClientFilter = filter.Chain{traces.ClientFilter(), tpsprometheus.ClientFilter()}.Handle
