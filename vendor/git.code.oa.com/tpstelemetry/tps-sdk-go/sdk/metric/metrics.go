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

// Package metric metric 子系统
package metric

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/automaxprocs/maxprocs"

	pkgruntime "git.code.oa.com/tpstelemetry/tps-sdk-go/pkg/runtime"
)

var (
	clientStartedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "client_started_total",
			Help:      "Total number of RPCs started on the client.",
		},
		// 当前服务名 要调用的服务名 要调用的rpc方法
		[]string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method"},
	)
	clientHandledCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "client_handled_total",
			Help:      "Total number of RPCs completed by the client, regardless of success or failure.",
		},
		[]string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method",
			"code", "code_type", "code_desc"},
	)
	clientHandledHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: "rpc",
			Name:      "client_handled_seconds",
			Help:      "Histogram of response latency (seconds) of the RPC until it is finished by the application.",
			Buckets:   []float64{.005, .01, .1, .5, 1, 5},
		},
		[]string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method"},
	)
)

var (
	serverStartedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "server_started_total",
			Help:      "Total number of RPCs started on the server.",
		},
		// rpc类型 调用方服务名 当前服务名 当前rpc方法
		[]string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method"},
	)
	serverHandledCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "server_handled_total",
			Help:      "Total number of RPCs completed on the server, regardless of success or failure.",
		},
		[]string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method",
			"code", "code_type", "code_desc"},
	)
	serverHandledHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: "rpc",
			Name:      "server_handled_seconds",
			Help:      "Histogram of response latency (seconds) of RPC that had been application-level handled by the server.",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 5},
		},
		[]string{"system_name", "caller_service", "caller_method", "callee_service", "callee_method"},
	)
)

var (
	// cpuCores 利用go.uber.org/automaxprocs获取正确的CPU数, 以便配合process_cpu_seconds_total指标拿到以100%为上限的CPU使用率
	cpuCores = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: "process",
			Name:      "cpu_cores",
			Help:      "Total CPU cores",
		},
	)
	// memoryQuota 通过cgroup获取被分配的内存总量，以便计算准确的内存使用率
	memoryQuota = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: "process",
			Name:      "memory_quota",
			Help:      "Total Memory Quota",
		},
	)
	// memoryUsage 通过cgroup获取使用的内存数，以便计算准确的内存使用率
	memoryUsage = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Subsystem: "process",
		Name:      "memory_usage",
		Help:      "Usage Memory",
	}, func() float64 {
		usageMemory, _ := pkgruntime.MemoryUsage()
		return float64(usageMemory)
	})
)

var (
	// ServerPanicTotal server panic count total
	ServerPanicTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "rpc",
			Name:      "server_panic_total",
			Help:      "Total number of RPCs panic on the server.",
		},
		[]string{"system_name"},
	)
)

const (
	rpcMetricsCardinalityLimit = 500
)

func init() {
	// client
	prometheus.MustRegister(&LimitCardinalityCollector{
		clientStartedCounter, "clientStartedCounter", rpcMetricsCardinalityLimit})
	prometheus.MustRegister(&LimitCardinalityCollector{
		clientHandledCounter, "clientHandledCounter", rpcMetricsCardinalityLimit})
	prometheus.MustRegister(&LimitCardinalityCollector{
		clientHandledHistogram, "clientHandledHistogram", rpcMetricsCardinalityLimit})
	// server
	prometheus.MustRegister(&LimitCardinalityCollector{
		serverStartedCounter, "serverStartedCounter", rpcMetricsCardinalityLimit})
	prometheus.MustRegister(&LimitCardinalityCollector{
		serverHandledCounter, "serverHandledCounter", rpcMetricsCardinalityLimit})
	prometheus.MustRegister(&LimitCardinalityCollector{
		serverHandledHistogram, "serverHandledHistogram", rpcMetricsCardinalityLimit})
	// system
	prometheus.MustRegister(cpuCores)
	prometheus.MustRegister(memoryQuota)
	prometheus.MustRegister(memoryUsage)
}

func init() {
	_, _ = maxprocs.Set(maxprocs.Logger(func(s string, i ...interface{}) {
	}))
	cpuCores.Set(float64(runtime.GOMAXPROCS(0)))
	totalMemory, _ := pkgruntime.MemoryQuota()
	memoryQuota.Set(float64(totalMemory))
}

// Code 错误码信息
type Code struct {
	// Code 错误码
	Code string `yaml:"code"`
	// Type 错误码类型
	Type string `yaml:"type"`
	// Description 错误码描述
	Description string `yaml:"description"`
	// Service 对应的service, 为空表示全匹配
	Service string `yaml:"service"`
	// Method 对应的method, 为空表示全匹配
	Method string `yaml:"method"`
}

// NewCode 创建一个错误码信息
func NewCode(code string, codeType CodeType, description string) *Code {
	// description 限制最大长度
	maxLength := 32
	if len(description) > maxLength {
		description = description[:maxLength]
	}
	// description作为Prometheus label需要utf8
	description = strings.ToValidUTF8(description, "")
	switch v := codeType; v {
	case CodeTypeSuccess, CodeTypeException, CodeTypeTimeout:
		return &Code{
			Code:        code,
			Type:        v.String(),
			Description: description,
		}
	default:
		return &Code{
			Code:        code,
			Type:        CodeTypeSuccess.String(),
			Description: description,
		}
	}
}

// String string 方法
func (c *Code) String() string {
	return fmt.Sprintf("%+v", *c)
}

// Deprecated: CodeTypeMappingDescription 描述code_type_mapping信息, 小结构体使用值类型
type CodeTypeMappingDescription struct {
	// CodeType 错误码类型
	CodeType string
	// Description 错误码描述
	Description string
}

// Deprecated: NewCodeTypeMappingDescription
func NewCodeTypeMappingDescription(codeType CodeType, description string) *CodeTypeMappingDescription {
	code := NewCode("", codeType, description)
	return &CodeTypeMappingDescription{
		CodeType:    code.Type,
		Description: code.Description,
	}
}

// UnmarshalText 实现自动从yaml string字段解析为*CodeTypeMappingDescription
func (d *CodeTypeMappingDescription) UnmarshalText(text []byte) error {
	seg := strings.Split(string(text), "|")
	var description string
	if len(seg) > 1 {
		description = seg[1]
	}
	*d = *NewCodeTypeMappingDescription(CodeType(seg[0]), description)
	return nil
}

// String string 方法
func (d *CodeTypeMappingDescription) String() string {
	return fmt.Sprintf("%+v", *d)
}

// CodeType 返回码
type CodeType string

const (
	CodeTypeSuccess   CodeType = "success"
	CodeTypeException CodeType = "exception"
	CodeTypeTimeout   CodeType = "timeout"
)

// String stirng 方法
func (c CodeType) String() string {
	s := string(c)
	if s == "" {
		return "success"
	}
	return s
}

// CodeTypeFunc CodeTypeFunc
type CodeTypeFunc func(code, service, method string) *Code

// DefaultCodeTypeFunc 默认code mapping
var DefaultCodeTypeFunc = defaultCodeTypeFunc

var (
	successCodeDesc = &Code{
		Type:        CodeTypeSuccess.String(),
		Description: "code=0",
	}
	exceptionCodeDesc = &Code{
		Type:        CodeTypeException.String(),
		Description: "code!=0",
	}
)

func defaultCodeTypeFunc(code, _, _ string) *Code {
	if code == "0" || code == "" {
		return successCodeDesc
	}
	return exceptionCodeDesc
}

// ClientReporter 客户端metrics上报
type ClientReporter struct {
	systemName    string
	callerService string
	callerMethod  string
	calleeService string
	calleeMethod  string
	startTime     time.Time
}

// NewClientReporter create a client reporter
func NewClientReporter(systemName, callerService, callerMethod, calleeService, calleeMethod string) *ClientReporter {
	r := &ClientReporter{
		systemName:    systemName,
		callerService: callerService,
		callerMethod:  CleanRPCMethod(callerMethod),
		calleeService: calleeService,
		calleeMethod:  CleanRPCMethod(calleeMethod),
		startTime:     time.Now(),
	}
	clientStartedCounter.WithLabelValues(r.systemName, r.callerService, r.callerMethod,
		r.calleeService, r.calleeMethod).Inc()
	return r
}

// Handled 在请求处理后调用.
func (r *ClientReporter) Handled(ctx context.Context, code string) {
	codeType := DefaultCodeTypeFunc(code, r.calleeService, r.calleeMethod)
	c := clientHandledCounter.WithLabelValues(r.systemName, r.callerService, r.callerMethod,
		r.calleeService, r.calleeMethod, code, codeType.Type, codeType.Description)
	h := clientHandledHistogram.WithLabelValues(r.systemName, r.callerService, r.callerMethod,
		r.calleeService, r.calleeMethod)
	sp := trace.SpanFromContext(ctx).SpanContext()
	if sp.IsSampled() && codeType.Type != CodeTypeSuccess.String() {
		exemplar := prometheus.Labels{
			"traceID": sp.TraceID().String(),
		}
		if v, ok := c.(prometheus.ExemplarAdder); ok {
			v.AddWithExemplar(1, exemplar)
		}
		if v, ok := h.(prometheus.ExemplarObserver); ok {
			v.ObserveWithExemplar(time.Since(r.startTime).Seconds(), exemplar)
		}
	} else {
		c.Inc()
		h.Observe(time.Since(r.startTime).Seconds())
	}
}

// ServerReporter 服务端metrics上报
type ServerReporter struct {
	systemName    string
	callerService string
	callerMethod  string
	calleeService string
	calleeMethod  string
	startTime     time.Time
}

// NewServerReporter create a server reporter
func NewServerReporter(systemName, callerService, callerMethod, calleeService, calleeMethod string) *ServerReporter {
	r := &ServerReporter{
		systemName:    systemName,
		callerService: formatServiceName(callerService), // 兼容调用方未在context中设置调用服务信息的场景 若未设置直接使用'-'填充
		callerMethod:  CleanRPCMethod(callerMethod),
		calleeService: formatServiceName(calleeService),
		calleeMethod:  CleanRPCMethod(calleeMethod),
		startTime:     time.Now(),
	}
	serverStartedCounter.WithLabelValues(r.systemName, r.callerService, r.callerMethod,
		r.calleeService, r.calleeMethod).Inc()
	return r
}

func formatServiceName(s string) string {
	if len(strings.TrimSpace(s)) == 0 {
		return "-"
	}
	return s
}

// Handled 在请求处理后调用.
func (r *ServerReporter) Handled(ctx context.Context, code string) {
	codeType := DefaultCodeTypeFunc(code, r.calleeService, r.calleeMethod)
	c := serverHandledCounter.WithLabelValues(r.systemName, r.callerService, r.callerMethod,
		r.calleeService, r.calleeMethod, code, codeType.Type, codeType.Description)
	h := serverHandledHistogram.WithLabelValues(r.systemName, r.callerService, r.callerMethod,
		r.calleeService, r.calleeMethod)
	sp := trace.SpanFromContext(ctx).SpanContext()
	if sp.IsSampled() && codeType.Type != CodeTypeSuccess.String() {
		exemplar := prometheus.Labels{
			"traceID": sp.TraceID().String(),
		}
		if v, ok := c.(prometheus.ExemplarAdder); ok {
			v.AddWithExemplar(1, exemplar)
		}
		if v, ok := h.(prometheus.ExemplarObserver); ok {
			v.ObserveWithExemplar(time.Since(r.startTime).Seconds(), exemplar)
		}
	} else {
		c.Inc()
		h.Observe(time.Since(r.startTime).Seconds())
	}
}

// CleanRPCMethod for high-cardinality problem
func CleanRPCMethod(method string) string {
	if method == "" {
		return "-"
	}
	if strings.HasPrefix(method, "/0x") {
		// oidb method
		return strings.ToValidUTF8(method, "")
	}
	if method[0] == '/' { // http path
		// 1. trim http query params (after char '?')
		if idx := strings.IndexByte(method, '?'); idx > 0 {
			method = method[:idx]
		}
		if v, ok := methodToPattern(method); ok {
			return strings.ToValidUTF8(v, "")
		}
		// http服务只信任通过RegisterMethodMapping的pattern, 避免高基数问题
		return "default_pattern_method"
	}
	// 3. limit length<64
	const maxLength = 64
	if len(method) > maxLength {
		method = method[:maxLength]
	}
	return strings.ToValidUTF8(method, "")
}

var methodMappings []*MethodMapping

// MethodMapping
type MethodMapping struct {
	Regex   *regexp.Regexp
	Pattern string
}

// RegisterMethodMapping 在初始化函数中注册 method regex->pattern 映射, 将含有 path 参数的 高基数method 转换为 低基数的 method pattern,
// regexStr 不合法将 Panic.
func RegisterMethodMapping(regexStr string, pattern string) {
	if !strings.HasPrefix(regexStr, "^") { // 添加完全匹配
		regexStr = "^" + regexStr + "$"
	}
	regex := regexp.MustCompile(regexStr)
	methodMappings = append(methodMappings, &MethodMapping{
		Regex:   regex,
		Pattern: pattern,
	})
}

// methodToPattern
func methodToPattern(method string) (string, bool) {
	if methodMappings == nil {
		return method, false
	}
	for _, v := range methodMappings {
		if v.Regex.MatchString(method) {
			return v.Pattern, true
		}
	}
	return method, false
}
