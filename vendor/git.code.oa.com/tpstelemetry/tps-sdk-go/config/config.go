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

// Package config 配置组件
package config

import (
	"strings"
	"time"

	tpstelemetry "git.code.oa.com/tpstelemetry/tps-sdk-go"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/api/log"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/metric"
)

// Config tpstelemetry trpc plugin config
type Config struct {
	Addr     string        `yaml:"addr"`
	TenantID string        `yaml:"tenant_id"`
	Sampler  SamplerConfig `yaml:"sampler"`
	Metrics  MetricsConfig `yaml:"metrics"`
	Logs     LogsConfig    `yaml:"logs"`
	Traces   TracesConfig  `yaml:"traces"`
}

// TracesConfig traces config
type TracesConfig struct {
	DisableTraceBody bool `yaml:"disable_trace_body"` // 若为true，则关闭trace中对req和rsp的上报，可以提高上报性能
	// EnableDeferredSample 是否开启延迟采样
	EnableDeferredSample bool `yaml:"enable_deferred_sample"`
	// DeferredSampleError 延迟采样-出错采样
	DeferredSampleError bool `yaml:"deferred_sample_error"`
	// DeferredSampleSlowDuration 延迟采样-慢操作采样
	DeferredSampleSlowDuration time.Duration `yaml:"deferred_sample_slow_duration"`
}

// SamplerConfig sampler config
type SamplerConfig struct {
	Fraction          float64       `yaml:"fraction"`
	SamplerServerAddr string        `yaml:"sampler_server_addr"`
	SyncInterval      time.Duration `yaml:"sync_interval"`
}

// MetricsConfig metrics config
type MetricsConfig struct {
	Enabled           bool     `yaml:"enabled"`
	RegistryEndpoints []string `yaml:"registry_endpoints"`
	ServerOwner       string   `yaml:"server_owner"`
	// Deprecated CodeTypeMapping codeType映射: key: code value: type(success/exception/timeout) 默认值:success
	CodeTypeMapping map[string]*metric.CodeTypeMappingDescription `yaml:"code_type_mapping"`
	Codes           []*metric.Code                                `yaml:"codes"`
}

// LogsConfig logs config
type LogsConfig struct {
	TraceLogMode    LogMode   `yaml:"trace_log_mode"`
	Level           log.Level `yaml:"level"`
	Enabled         bool      `yaml:"enabled"`
	DisableRecovery bool      `yaml:"disable_recovery"` // tpstelemetry log filter默认会recovery panic并打印日志上报指标, 可以关闭此行为
	EnableSampler   bool      `yaml:"enable_sampler"`   // 启用采样器, 只有当前请求命中采样时才会上报独立日志
	RateLimit       RateLimit `yaml:"rate_limit"`       // 日志限频配置
}

// RateLimit 日志流控配置
// 例如，tick = 1s，first = 100, thereafter = 3 表示1秒内同一条日志打印超过100条后，则每隔3条才打印这一条相同的日志
// 这里定义"相同的日志"为内容和等级都完全相同的重复日志
type RateLimit struct {
	EnableRateLimit bool `yaml:"enable_rate_limit"` // 是否开启日志流控配置
	// tick 日志流控的生效周期（即从打印一条日志开始计时在tick时间后，无论触发限流与否，对同一条计数器会被置为零，重新开始计数)
	Tick       time.Duration `yaml:"tick"`
	First      int           `yaml:"first"`      // first 限流阈值，即相同的日志达到first条时触发限流
	Thereafter int           `yaml:"thereafter"` // thereafter  触发限流后每thereafter条相同日志才会输出一条
}

// LogMode 日志级别
type LogMode int32

const (
	LogModeDefault   LogMode = 0 // default
	LogModeOneLine   LogMode = 1
	LogModeDisable   LogMode = 2
	LogModeMultiLine LogMode = 3
)

// DefaultConfig 默认配置
func DefaultConfig() Config {
	cfg := Config{
		Addr:     "localhost:12520",
		TenantID: tpstelemetry.DefaultTenantID,
		Sampler: SamplerConfig{
			Fraction:          0.001,
			SamplerServerAddr: "localhost:14941",
		},
		Metrics: MetricsConfig{
			Enabled:           true,
			RegistryEndpoints: []string{"localhost:2379"},
		},
		Logs: LogsConfig{
			Enabled:      false,
			TraceLogMode: LogModeDisable,
		},
		Traces: TracesConfig{DisableTraceBody: false},
	}
	return cfg
}

var logModeMap = map[string]LogMode{
	logModeStrDisable:   LogModeDisable,   // 不打印
	logModeStrVerbose:   LogModeOneLine,   // 单行包括包体
	logModeStrDefault:   LogModeOneLine,   // 默认值
	logModeStrMultiLine: LogModeMultiLine, // 多行
	logModeStrOneLine:   LogModeOneLine,   // 单行
}

// UnmarshalText LogMode 解析
func (m *LogMode) UnmarshalText(text []byte) error {
	switch v := logModeMap[strings.ToLower(string(text))]; v {
	case LogModeDisable, LogModeOneLine, LogModeMultiLine:
		*m = v
		return nil
	default:
		v = LogModeOneLine
		*m = v
		return nil
	}
}

const (
	logModeStrDisable   = "disable"
	logModeStrVerbose   = "verbose"
	logModeStrDefault   = ""
	logModeStrMultiLine = "multiline"
	logModeStrOneLine   = "oneline"
)

var logModeReverseMap = map[LogMode]string{
	LogModeDisable:   logModeStrDisable,   // 不打印
	LogModeOneLine:   logModeStrOneLine,   // 单行包括包体
	LogModeMultiLine: logModeStrMultiLine, // 多行
}

// MarshalText MarshalText
func (m LogMode) MarshalText() (text []byte, err error) {
	switch v := logModeReverseMap[m]; v {
	case logModeStrDisable, logModeStrOneLine, logModeStrMultiLine:
		return []byte(v), nil
	default:
		return []byte(logModeStrOneLine), nil
	}
}
