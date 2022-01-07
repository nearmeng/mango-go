package pcgmonitor

import (
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/sample"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/stat"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

// ReportAttr [框架]属性上报, 全策略
func (s *Instance) ReportAttr(name string, value float64) error {
	if s.svrInfoType != svrInfoFrame {
		return ErrorReportNotInit
	}

	c := s.remoteConfig()
	if !sample.Pass(c.attaInfo.SamplingRate) {
		return nil
	}

	statValues := make([]*nmnt.StatValue, 0, len(attrPolicies))
	for _, policy := range attrPolicies {
		statValue := &nmnt.StatValue{
			Value:  value,
			Count:  1,
			Policy: policy,
		}
		statValues = append(statValues, statValue)
	}

	statContent := &nmnt.StatContent{
		Dimensions: s.staticDimensions(1),
		Values:     statValues,
		Time:       time.Now().UnixNano() / 1e6,
	}
	logName := s.getLogName(c.prefixInfo.prefixMetrics)
	dimension := stat.SingleDimensionFix(logName, name, c.details, 7)
	statContent.Dimensions = append(statContent.Dimensions, dimension)

	return s.withStatLogCh(logName, statContent, metricsReport)
}
