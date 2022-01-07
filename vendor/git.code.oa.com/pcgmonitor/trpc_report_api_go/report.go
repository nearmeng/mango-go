package pcgmonitor

import (
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

// Setup [框架]初始化
func Setup(info *FrameSvrSetupInfo) error {
	return defaultInstance.Setup(info)
}

// ReportActive [框架]主调上报
func ReportActive(activeMsg *ActiveMsg) error {
	return defaultInstance.ReportActive(activeMsg)
}

// ReportPassive [框架]被调上报
func ReportPassive(passiveMsg *PassiveMsg) error {
	return defaultInstance.ReportPassive(passiveMsg)
}

// ReportAttr [框架]属性上报, 全策略
func ReportAttr(name string, value float64) error {
	return defaultInstance.ReportAttr(name, value)
}

// ReportCustom [框架]自定义数据上报
func ReportCustom(customItemName string, dimensions []string, statValues []*nmnt.StatValue) error {
	return defaultInstance.ReportCustom(customItemName, dimensions, statValues)
}

// CommSetUp [非框架]初始化
func CommSetUp(commServerInfo *CommSvrSetupInfo) error {
	return defaultInstance.CommSetUp(commServerInfo)
}

// CommReport [非框架]自定义上报
func CommReport(dimensions []string, statValues []*nmnt.StatValue) error {
	return defaultInstance.CommReport(dimensions, statValues)
}

// CommReport [非框架]自定义上报, 支持实例上报不同监控项
func CommReportWithSuffix(commItemName string, dimensions []string, statValues []*nmnt.StatValue) error {
	return defaultInstance.CommReportWithSuffix(commItemName, dimensions, statValues)
}

// ReportBusiness [框架]&[非框架]业务数据告警上报
func ReportBusiness(msg *BusinessMsg) error {
	return defaultInstance.ReportBusiness(msg)
}

// ReportHawkLog [框架]&[非框架] 业务日志上报
func ReportHawkLog(params *HawkLogParams) error {
	return defaultInstance.ReportHawkLog(params)
}

// AddLogNames 添加日志名称
func AddLogNames(logNames []string) {
	defaultInstance.AddLogNames(logNames)
}
