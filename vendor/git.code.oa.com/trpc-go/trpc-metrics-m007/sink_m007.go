package m007

import (
	"errors"

	pcgmonitor "git.code.oa.com/pcgmonitor/trpc_report_api_go"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
	"git.code.oa.com/trpc-go/trpc-go/metrics"
)

const (
	m007SinkName = "m007Sink"
)

var (
	policyMap = map[metrics.Policy]nmnt.Policy{
		metrics.PolicySET: nmnt.Policy_SET,
		metrics.PolicySUM: nmnt.Policy_SUM,
		metrics.PolicyAVG: nmnt.Policy_AVG,
		metrics.PolicyMAX: nmnt.Policy_MAX,
		metrics.PolicyMIN: nmnt.Policy_MIN,
	}
)

// M007Sink 实现pcgmonitor collector，对接公司pcg_monitor指标监控系统
type M007Sink struct{}

// Name name
func (m *M007Sink) Name() string {
	return m007SinkName
}

// Report 数据上报，适配框架上报接口
func (m *M007Sink) Report(rec metrics.Record, opts ...metrics.Option) error {
	if len(rec.GetDimensions()) <= 0 {
		return singleAttrReport(rec)
	}

	return multiAttrReport(rec)
}

// multiAttrReport 多维度上报
func multiAttrReport(rec metrics.Record) error {
	var dimesions []string
	var statValues []*nmnt.StatValue
	for _, t := range rec.GetDimensions() {
		dimesions = append(dimesions, t.Value)
	}
	for _, t := range rec.GetMetrics() {
		policy, ok := policyMap[t.Policy()]
		if !ok {
			return errors.New("policy 007 not support")
		}
		statValues = append(statValues, &nmnt.StatValue{Value: t.Value(), Count: 1, Policy: policy})
	}
	_ = pcgmonitor.ReportCustom(rec.Name, dimesions, statValues)
	return nil
}

// singleAttrReport 属性上报
func singleAttrReport(rec metrics.Record) error {
	for _, metric := range rec.GetMetrics() {
		_ = pcgmonitor.ReportAttr(metric.Name(), metric.Value()) // 007属性全策略上报
	}
	return nil
}
