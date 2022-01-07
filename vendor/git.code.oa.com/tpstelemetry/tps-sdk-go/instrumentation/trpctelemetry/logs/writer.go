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
	"errors"

	v1proto "github.com/golang/protobuf/proto"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/plugin"

	tpstelemetry "git.code.oa.com/tpstelemetry/tps-sdk-go"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/api"
	logtps "git.code.oa.com/tpstelemetry/tps-sdk-go/api/log"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/config"
	otlplog "git.code.oa.com/tpstelemetry/tps-sdk-go/exporter/otlp"
	tpsprometheus "git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/metrics/prometheus"
	sdklog "git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/log"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/tpszap"
)

const (
	typeStr = "tpstelemetry"
)

func init() {
	log.RegisterWriter(typeStr, &writer{})
}

var _ plugin.Factory = (*writer)(nil)

type writer struct {
}

// Type Type
func (w writer) Type() string {
	return typeStr
}

// Setup 启动配置
func (w writer) Setup(name string, configDec plugin.Decoder) error {
	decoder, ok := configDec.(*log.Decoder)
	if !ok {
		return errors.New("log decoder type invalid")
	}
	cfg := &config.Config{
		Addr:     tpstelemetry.DefaultExporterAddr,
		TenantID: tpstelemetry.DefaultTenantID,
		Logs: config.LogsConfig{
			Enabled: false,
			Level:   tpstelemetry.DefaultLogLevel,
		},
	}
	telemetrys, ok := trpc.GlobalConfig().Plugins["telemetry"]
	if ok {
		tpsDec, ok := telemetrys["tpstelemetry"]
		if ok {
			err := tpsDec.Decode(cfg)
			if err != nil {
				return errors.New("tpstelemetry config decode fail: " + err.Error())
			}
		}
	}

	if decoder.OutputConfig.Level != "" {
		var s logtps.Level
		err := s.UnmarshalText([]byte(decoder.OutputConfig.Level))
		if err != nil {
			return errors.New("tpstelemetry level invalid: " + decoder.OutputConfig.Level)
		}

		cfg.Logs.Level = s
	}

	if !cfg.Logs.Enabled {
		decoder.Core = zapcore.NewNopCore()
		return nil
	}

	exp, err := otlplog.NewExporter(otlplog.WithInsecure(),
		otlplog.WithAddress(cfg.Addr),
		otlplog.WithCompressor("gzip"),
		otlplog.WithHeaders(map[string]string{api.TenantHeaderKey: cfg.TenantID}),
		otlplog.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(
			grpc_prometheus.UnaryClientInterceptor,
			packetLogSizeMetric(),
		)))
	if err != nil {
		return errors.New("tpstelemetry log exporter create fail: " + err.Error())
	}

	kvs := []attribute.KeyValue{
		api.TpsTenantIDKey.String(cfg.TenantID),
		attribute.Key("server").String(trpc.GlobalConfig().Server.App + "." + trpc.GlobalConfig().Server.Server),
		attribute.Key("env").String(trpc.GlobalConfig().Global.EnvName),
	}

	var opts []sdklog.LoggerOption
	if cfg.Logs.Level != "" {
		opts = append(opts, sdklog.WithLevelEnable(cfg.Logs.Level))
	}
	opts = append(opts, sdklog.WithEnableSampler(cfg.Logs.EnableSampler))

	decoder.Core = tpszap.NewBatchCore(tpszap.NewBatchWriteSyncer(exp, resource.NewWithAttributes(kvs...),
		tpszap.WithEnabelSampler(cfg.Logs.EnableSampler)), opts...)

	if enableLogRateLimit(cfg) {
		decoder.Core = zapcore.NewSamplerWithOptions(decoder.Core,
			cfg.Logs.RateLimit.Tick, cfg.Logs.RateLimit.First, cfg.Logs.RateLimit.Thereafter)
	}

	log.Info("tsptelemetry zap log setup success")
	return nil
}

func enableLogRateLimit(cfg *config.Config) bool {
	if cfg == nil || !cfg.Logs.RateLimit.EnableRateLimit {
		return false
	}
	return cfg.Logs.RateLimit.First != 0 && cfg.Logs.RateLimit.Thereafter != 0 && cfg.Logs.RateLimit.Tick != 0
}

// packetLogSizeMetric log的包大小监控
func packetLogSizeMetric() func(ctx context.Context, method string,
	req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		switch req := req.(type) {
		case proto.Message:
			tpsprometheus.ObserveExportLogsBytes(proto.Size(req))
		case v1proto.Message:
			tpsprometheus.ObserveExportLogsBytes(v1proto.Size(req))
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
