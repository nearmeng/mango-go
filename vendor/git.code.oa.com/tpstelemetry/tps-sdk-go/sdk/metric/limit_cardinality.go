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

// Package metric metric 子系统
package metric

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

// highCardinalityMetrics 记录高基数的metrics, 进行告警
var highCardinalityMetrics = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "high_cardinality_metrics",
	Help: "high cardinality metrics",
}, []string{"name"})

var (
	// PerMetricCardinalityLimit 单个指标基数限制
	// ref: https://prometheus.io/docs/practices/naming/#labels
	// https://prometheus.io/docs/prometheus/latest/querying/basics/#avoiding-slow-queries-and-overloads
	PerMetricCardinalityLimit = 2000
	// TotalMetricCardinalityLimit 服务实例总指标基数限制
	TotalMetricCardinalityLimit = 10000
)

// LimitMetricsHandler 限制metrics个数的handler, 避免Prometheus server OOM
func LimitMetricsHandler() http.Handler {
	gather := &LimitCardinalityGatherer{
		prometheus.DefaultGatherer,
		PerMetricCardinalityLimit,
		TotalMetricCardinalityLimit}
	return promhttp.InstrumentMetricHandler(
		prometheus.DefaultRegisterer, promhttp.HandlerFor(gather, promhttp.HandlerOpts{
			// OpenMetrics is the only way to transmit exemplars. However, the move to OpenMetrics
			// is not completely transparent. Most notably, the values of "quantile"
			// labels of Summaries and "le" labels of Histograms are formatted with
			// a trailing ".0" if they would otherwise look like integer numbers
			// (which changes the identity of the resulting series on the Prometheus
			// server).
			EnableOpenMetrics: true,
		}),
	)
}

// LimitCardinalityGatherer 带限制的收集器
type LimitCardinalityGatherer struct {
	prometheus.Gatherer
	PerMetirclimit   int
	TotalMetricLimit int
}

// Gather implements prometheus.Gatherer
func (l *LimitCardinalityGatherer) Gather() ([]*dto.MetricFamily, error) {
	res, err := l.Gatherer.Gather()
	if err != nil {
		return nil, err
	}
	var total int
	var resetClientM, resetServerM sync.Once
	for i, v := range res {
		if l.PerMetirclimit > 0 && len(v.GetMetric()) > l.PerMetirclimit {
			log.Printf("tpstelemetry: high cardinality metric '%s', value:%d, limit:%d",
				v.GetName(), len(v.GetMetric()), l.PerMetirclimit)
			highCardinalityMetrics.WithLabelValues(v.GetName()).Set(float64(len(v.GetMetric())))
			// 降级处理, 取topN
			v.Metric = v.Metric[:l.PerMetirclimit]
			// rpc系列指标高基数, reset清空处理, 非频繁reset不会影响Counter增量计算
			if strings.HasPrefix(v.GetName(), "rpc_client") {
				resetClientM.Do(func() {
					log.Printf("tpstelemetry: reset rpc_client metric when high cardinality(>%d)",
						l.PerMetirclimit)
					clientStartedCounter.Reset()
					clientHandledCounter.Reset()
					clientHandledHistogram.Reset()
				})
			}
			if strings.HasPrefix(v.GetName(), "rpc_server") {
				resetServerM.Do(func() {
					log.Printf("tpstelemetry: reset rpc_server metric when high cardinality(>%d)",
						l.PerMetirclimit)
					serverStartedCounter.Reset()
					serverHandledCounter.Reset()
					serverHandledHistogram.Reset()
				})
			}
		}
		total += len(v.GetMetric())
		if l.TotalMetricLimit > 0 && total > l.TotalMetricLimit {
			log.Printf("tpstelemetry: high cardinality metric '%s', value:%d %d, limit:%d",
				"all", total, len(res), l.TotalMetricLimit)
			highCardinalityMetrics.WithLabelValues("total").Set(float64(len(res)))
			// 降级处理, 只保留部分
			return res[:i], nil
		}
	}
	return res, nil
}

type metricCollector interface {
	prometheus.Collector
	Delete(labels prometheus.Labels) bool
}

// LimitCardinalityCollector 限制高基数Metrics的包装器
type LimitCardinalityCollector struct {
	metricCollector
	desc  string
	limit int
}

// Collect 会在每次pull metrics时调用
func (c *LimitCardinalityCollector) Collect(ch chan<- prometheus.Metric) {
	results := c.collect()
	if len(results) <= c.limit {
		// fast path 未超过限制
		// 回写结果
		for _, v := range results {
			ch <- v
		}
		return
	}
	// 超出限制
	metrics := map[*dto.Metric]prometheus.Metric{}
	var m []*dto.Metric
	for _, v := range results {
		mm := &dto.Metric{}
		_ = v.Write(mm)
		m = append(m, mm)
		metrics[mm] = v
	}
	if len(m) > c.limit {
		highCardinalityMetrics.WithLabelValues(c.desc).Set(float64(len(m)))
		log.Printf("tpstelemetry: metric '%s' high cardinality, limit:%d", c.desc, c.limit)
		// 降级处理: 删除值小于等于1的指标维度, 删除值最小的 len(m)-c.limit 个指标维度
		sort.Slice(m, func(i, j int) bool {
			return getMetricValue(m[i]) >
				getMetricValue(m[j])
		})
		for i := 0; i < len(m); i++ {
			shouldDelete := getMetricValue(m[i]) <= 1 || i >= c.limit
			if shouldDelete {
				labels := prometheus.Labels{}
				for _, vv := range m[i].GetLabel() {
					labels[vv.GetName()] = vv.GetValue()
				}
				c.Delete(labels)
				delete(metrics, m[i])
			}
		}
	}
	// 回写结果
	for _, v := range metrics {
		ch <- v
	}
}

func (c *LimitCardinalityCollector) collect() []prometheus.Metric {
	tmpCh := make(chan prometheus.Metric, 10)
	done := make(chan struct{})
	go func() {
		c.metricCollector.Collect(tmpCh)
		close(done)
	}()
	go func() {
		<-done
		close(tmpCh)
	}()
	var results []prometheus.Metric
	for v := range tmpCh {
		results = append(results, v)
	}
	return results
}

func getMetricValue(m *dto.Metric) float64 {
	return m.GetCounter().GetValue() +
		m.GetGauge().GetValue() +
		float64(m.GetHistogram().GetSampleCount()) +
		float64(m.GetSummary().GetSampleCount()) +
		m.GetUntyped().GetValue()
}
