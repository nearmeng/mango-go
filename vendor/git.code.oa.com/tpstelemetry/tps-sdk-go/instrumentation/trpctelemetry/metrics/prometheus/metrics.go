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

// Package prometheus prometheus metrics
package prometheus

import (
	"git.code.oa.com/trpc-go/trpc-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// tpstelemetrySDKPanicTotal sdk panic 监控指标
	tpstelemetrySDKPanicTotal = promauto.NewCounter(prometheus.CounterOpts{
		Subsystem: "tpstelemetry_sdk",
		Name:      "panic_total",
		Help:      "tpstelemetry sdk panic total",
	})

	// exportSpansBytes 上报字节数监控指标
	exportSpansBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: "tpstelemetry_sdk",
		Name:      "export_spans_bytes",
		Help:      "Export Spans Bytes",
		Buckets:   []float64{128, 1024, 10240, 102400, 4194304},
	})

	// exportLogsBytes 上报logs字节数监控指标
	exportLogsBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: "tpstelemetry_sdk",
		Name:      "export_logs_bytes",
		Help:      "Export Logs Bytes",
		Buckets:   []float64{128, 1024, 10240, 102400, 4194304},
	})

	// requestBodyBytes 上报网络请求包体监控指标
	requestBodyBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: "rpc",
		Name:      "request_body_bytes",
		Help:      "Request Body Bytes",
		Buckets:   []float64{1024, 10240, 102400, 1024_000, 10240_000},
	})

	// requestMetaDataBodyBytes 上报请求matadata大小监控指标
	requestMetaDataBodyBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: "rpc",
		Name:      "request_matadata_bytes",
		Help:      "Request Metadata Bytes",
		Buckets:   []float64{1024, 10240, 102400, 1024_000, 10240_000},
	})

	// trpcSDKMetadata 上报trpc相关sdk版本信息指标
	trpcSDKMetadata = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tpstelemetry_trpc_metadata",
		Help: "tpstelemetry trpc metadata version",
	}, []string{
		"trpc_version",
	})
)

// IncrSDKPanicTotal sdk panic数指标
func IncrSDKPanicTotal() {
	tpstelemetrySDKPanicTotal.Inc()
}

// ObserveExportSpansBytes span 大小指标
func ObserveExportSpansBytes(s int) {
	exportSpansBytes.Observe(float64(s))
}

// ObserveExportLogsBytes log 大小指标
func ObserveExportLogsBytes(s int) {
	exportLogsBytes.Observe(float64(s))
}

// ObserveRequestBodyBytes 监控请求包体大小指标
func ObserveRequestBodyBytes(s int) {
	requestBodyBytes.Observe(float64(s))
}

// ObserveRequestMataDataBytes 监控metadata 大小的指标
func ObserveRequestMataDataBytes(s int) {
	requestMetaDataBodyBytes.Observe(float64(s))
}

// MonitorTRPCSDKMeta trpc sdk 上报指标
func MonitorTRPCSDKMeta() {
	trpcSDKMetadata.WithLabelValues(trpc.Version()).Set(1)
}
