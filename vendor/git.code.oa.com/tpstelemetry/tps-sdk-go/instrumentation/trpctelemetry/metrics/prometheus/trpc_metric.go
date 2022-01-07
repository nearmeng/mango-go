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
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/metrics"
	"github.com/mozillazg/go-pinyin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

func init() {
	metrics.RegisterMetricsSink(NewSink(prometheus.DefaultRegisterer))
}

// Sink 实现 trpc metrics.Sink接口, 将使用 trpc metrics API的指标转换为Prometheus格式.
// 转换规则与trpc-metrics-prometheus插件不同:
// trpc metric name => 由于Prometheus name不支持中文及特殊字符, 将原始的name存储在 `_name` label中, name使用转换的拼音名字
// trpc metric type => prometheus `_type` label
// 2021-09-10 update: 由于使用trpc metrics命名过长的情况太多, 一一映射为拼音指标名去承载会使得Prometheus metadata api失去作用.
// 改为每种类型使用同一个指标名. 用不同label来承载. 这对于Dashboard中使用{_name="xx"}这种语句来展示是兼容的.
type Sink struct {
	counters   *prometheus.CounterVec
	gauges     *prometheus.GaugeVec
	histograms sync.Map
	registerer prometheus.Registerer
}

// NewSink create a new Sink
func NewSink(registerer prometheus.Registerer) *Sink {
	s := &Sink{
		registerer: registerer,
	}
	s.counters = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "trpc_counter_total", // counter命名须以_total结尾
		Help: "trpc metrics counter",
	},
		[]string{"_name", "_type"},
	).MustCurryWith(prometheus.Labels{"_type": "counter"})
	s.gauges = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trpc_gauge",
		Help: "trpc metrics gauge",
	},
		[]string{"_name", "_type"},
	).MustCurryWith(prometheus.Labels{"_type": "gauge"})
	_ = s.registerer.Register(s.counters)
	_ = s.registerer.Register(s.gauges)
	return s
}

// Name implements metrics.Sink.
func (*Sink) Name() string {
	return "tpstelemetry"
}

// Report implements metrics.Sink. 会在每次metrics操作时同步调用
func (s *Sink) Report(rec metrics.Record, opts ...metrics.Option) error {
	for _, m := range rec.GetMetrics() {
		switch m.Policy() {
		case metrics.PolicySUM:
			s.incrCounter(m.Name(), m.Value())
		case metrics.PolicySET:
			s.setGauge(m.Name(), m.Value())
		case metrics.PolicyTimer:
			s.observeHistogram(m.Name(), time.Duration(int64(m.Value())).Seconds())
		case metrics.PolicyHistogram:
			s.observeHistogram(m.Name(), m.Value())
		case metrics.PolicyAVG, metrics.PolicyMAX, metrics.PolicyMIN, metrics.PolicyMID:
			s.setGauge(m.Name(), m.Value())
		default:
			s.setGauge(m.Name(), m.Value())
		}
	}
	return nil
}

func (s *Sink) register(metric interface{}, name string) {
	c, ok := metric.(prometheus.Collector)
	if !ok {
		return
	}
	if err := s.registerer.Register(c); err != nil {
		log.Warnf("tpstelemetry: register err:%v, metric:%s", err, name)
	}
}

func (s *Sink) incrCounter(name string, value float64) {
	s.counters.WithLabelValues(strings.ToValidUTF8(name, "")).Add(value)
}

func (s *Sink) setGauge(name string, value float64) {
	s.gauges.WithLabelValues(strings.ToValidUTF8(name, "")).Set(value)
}

// observeHistogram 不同的Histogram的桶可能不同, 所以只能动态创建
func (s *Sink) observeHistogram(name string, value float64) {
	var m interface{}
	var ok bool
	if m, ok = s.histograms.Load(name); !ok {
		var buckets []float64
		if h, ok2 := metrics.GetHistogram(name); ok2 {
			for _, b := range h.GetBuckets() {
				buckets = append(buckets, b.ValueUpperBound)
			}
		} else {
			buckets = prometheus.DefBuckets
		}
		m = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "trpc_histogram_" + s.convertMetricName(name),
			Help:    strings.ToValidUTF8(name, ""),
			Buckets: buckets,
		},
			[]string{"_name", "_type"},
		).WithLabelValues(strings.ToValidUTF8(name, ""), "histogram")
		if m, ok = s.histograms.LoadOrStore(name, m); !ok {
			s.register(m, name)
		}
	}
	m.(prometheus.Histogram).Observe(value)
}

// convertMetricName 如果不符合Prometheus metric name, 则转换为拼音, 特殊字符用_或hex表示
func (s *Sink) convertMetricName(origin string) string {
	if model.LabelName(origin).IsValid() {
		return origin
	}
	var buf strings.Builder
	for i, r := range origin {
		switch {
		case isNormalChar(i, r):
			buf.WriteRune(r)
		case unicode.Is(unicode.Han, r):
			// 汉字转拼音
			p := pinyin.NewArgs()
			buf.WriteString(strings.Join(pinyin.SinglePinyin(r, p), "_"))
			buf.WriteRune('_')
		case r < utf8.RuneSelf:
			// 特殊字符替换为_
			buf.WriteRune('_')
		default:
			// 其它特殊字符使用hex
			buf.WriteString(fmt.Sprintf("%X", r))
			buf.WriteRune('_')
		}
	}
	result := strings.TrimSuffix(buf.String(), "_")
	return result
}

func isNum(b rune) bool {
	return b >= '0' && b <= '9'
}

func isChar(b rune) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// 字母或数字或下划线, 首字符不为数字
func isNormalChar(i int, b rune) bool {
	return isChar(b) || b == '_' || (isNum(b) && i > 0)
}
