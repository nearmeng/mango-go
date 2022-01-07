package pcgmonitor

import (
	"strings"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

// getLogName 拼接logName，细化到服务粒度
func (s *Instance) getLogName(prefix string) string {
	var b strings.Builder
	b.WriteString(prefix)
	b.WriteByte('_')
	if s.svrInfoType == svrInfoFrame {
		b.WriteString(s.frameSvrInfo.App)
		b.WriteByte('.')
		b.WriteString(s.frameSvrInfo.Server)
	} else {
		b.WriteString(s.commSvrInfo.CommName)
	}
	return b.String()
}

// staticDimensions 监控上报项静态数据，服务相关属性
func (s *Instance) staticDimensions(externalSize int) []string {
	staticDimensions := make([]string, 0, externalSize+7)

	staticDimensions = append(staticDimensions, s.frameSvrInfo.App+"."+s.frameSvrInfo.Server)
	staticDimensions = append(staticDimensions, s.frameSvrInfo.IP)
	staticDimensions = append(staticDimensions, s.frameSvrInfo.Container)
	staticDimensions = append(staticDimensions, s.frameSvrInfo.ConSetId)
	staticDimensions = append(staticDimensions, s.frameSvrInfo.Version)
	staticDimensions = append(staticDimensions, s.frameSvrInfo.PhysicEnv)
	staticDimensions = append(staticDimensions, s.frameSvrInfo.UserEnv)

	return staticDimensions
}

// withStatLogCh 将数据写到管道，异步定时上报
func (s *Instance) withStatLogCh(logName string, statContent *nmnt.StatContent, reportType reportType) error {
	statLog := &nmnt.StatLog{
		Logname: logName,
		Content: []*nmnt.StatContent{statContent},
	}

	ret := 0
	select {
	case s.channel <- statLog:
	default:
		ret = 1
		select {
		case s.reportLooper.active <- 1: // 管道满，数据丢弃，兜底激活goroutinue上报数据
		default:
		}
	}

	s.sendDependOnContentNum()
	s.selfMonitorLooper.addSendReportSelfMonitor(ret, reportType, logName)

	if ret != 0 {
		return ErrorReportLossData
	}
	return nil
}

// checkStrInSlice 检测切片中是否包含一个字符串元素
func checkStrInSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// version 返回api版本
func version() string {
	return "3.13"
}
