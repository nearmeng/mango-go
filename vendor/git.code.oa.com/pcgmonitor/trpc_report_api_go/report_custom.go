package pcgmonitor

import (
	"fmt"
	"log"
	"strings"
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/sample"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/stat"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

// ReportCustom [框架]自定义数据上报
func (s *Instance) ReportCustom(customItemName string, dimensions []string, statValues []*nmnt.StatValue) error {
	if s.svrInfoType != svrInfoFrame {
		return ErrorReportNotInit
	}

	c := s.remoteConfig()
	if !sample.Pass(c.attaInfo.SamplingRate) {
		return nil
	}

	stat.ValueFix(statValues)

	var b strings.Builder
	b.WriteString(s.getLogName(c.prefixInfo.prefixCustom))
	b.WriteByte('_')
	b.WriteString(customItemName)
	logName := b.String()
	stat.MultiDimensionsFix(logName, dimensions, c.details, 7)

	statContent := &nmnt.StatContent{
		Dimensions: s.staticDimensions(len(dimensions)),
		Values:     statValues,
		Time:       time.Now().UnixNano() / 1e6,
	}
	statContent.Dimensions = append(statContent.Dimensions, dimensions...)
	// 数据写管道，异步上报
	return s.withStatLogCh(logName, statContent, customReport)
}

// CommReport [非框架]自定义上报
func (s *Instance) CommReport(dimensions []string, statValues []*nmnt.StatValue) error {
	if s.svrInfoType != svrInfoComm {
		return ErrorReportNotInit
	}

	c := s.remoteConfig()
	// 采样
	if !sample.Pass(c.attaInfo.SamplingRate) {
		return nil
	}

	stat.ValueFix(statValues)

	logName := s.getLogName(c.prefixInfo.prefix)
	stat.MultiDimensionsFix(logName, dimensions, c.details, 0)

	statContent := &nmnt.StatContent{
		Dimensions: dimensions,
		Values:     statValues,
		Time:       time.Now().UnixNano() / 1e6,
	}
	return s.withStatLogCh(logName, statContent, commonReport)
}

// CommReport [非框架]自定义上报, 支持单实例上报不同监控项
func (s *Instance) CommReportWithSuffix(commItemName string, dimensions []string, statValues []*nmnt.StatValue) error {
	if s.svrInfoType != svrInfoComm {
		return ErrorReportNotInit
	}

	c := s.remoteConfig()
	// 采样
	if !sample.Pass(c.attaInfo.SamplingRate) {
		return nil
	}

	stat.ValueFix(statValues)

	var b strings.Builder
	b.WriteString(s.getLogName(c.prefixInfo.prefix))
	b.WriteByte('_')
	b.WriteString(commItemName)
	logName := b.String()
	stat.MultiDimensionsFix(logName, dimensions, c.details, 0)

	statContent := &nmnt.StatContent{
		Dimensions: dimensions,
		Values:     statValues,
		Time:       time.Now().UnixNano() / 1e6,
	}
	return s.withStatLogCh(logName, statContent, commonReport)
}

// BusinessLevelType 业务数据告警日志级别
type BusinessLevelType int

const (
	BusinessLevelDebug   BusinessLevelType = 2
	BusinessLevelWarning BusinessLevelType = 3
	BusinessLevelError   BusinessLevelType = 4
	BusinessLevelFetal   BusinessLevelType = 5
)

// BusinessMsg 业务数据告警，字段命名&类型遵守：https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=90370975
type BusinessMsg struct {
	IP      string            // 用户IP
	QQ      int64             // 用户QQ
	biz     string            // 业务模块名
	Op      string            // 操作名称
	Status  int64             // 错误码
	Type    int64             // 日志类型
	Flow    int64             // 流水ID
	SrcFile string            // 源文件名
	SrcLine int64             // 行号
	Func    string            // 函数名
	PName   string            // 进程名
	PID     int64             // 进程ID
	Level   BusinessLevelType // 错误级别
	ErrMsg  string            // 错误内容
}

// fixFromInstance 部分字段补齐
func (msg *BusinessMsg) fixFromInstance(s *Instance) {
	// 设置模块名
	if s.svrInfoType == svrInfoFrame {
		msg.biz = fmt.Sprintf("%s.%s", s.frameSvrInfo.App, s.frameSvrInfo.Server)

		if msg.IP == "" {
			msg.IP = s.frameSvrInfo.IP
		}
	} else {
		msg.biz = s.commSvrInfo.CommName

		if msg.IP == "" {
			msg.IP = s.commSvrInfo.IP
		}
	}
}

// ReportBusiness [框架]&[非框架]业务数据告警上报
func (s *Instance) ReportBusiness(msg *BusinessMsg) error {
	if s.svrInfoType != svrInfoFrame && s.svrInfoType != svrInfoComm {
		return ErrorReportNotInit
	}

	msg.fixFromInstance(s)

	// 按顺序拼接fields
	fields := make([]string, 0, 14)
	fields = append(fields, msg.IP)
	fields = append(fields, fmt.Sprintf("%d", msg.QQ))
	fields = append(fields, msg.biz)
	fields = append(fields, msg.Op)
	fields = append(fields, fmt.Sprintf("%d", msg.Status))
	fields = append(fields, fmt.Sprintf("%d", msg.Type))
	fields = append(fields, fmt.Sprintf("%d", msg.Flow))
	fields = append(fields, msg.SrcFile)
	fields = append(fields, fmt.Sprintf("%d", msg.SrcLine))
	fields = append(fields, msg.Func)
	fields = append(fields, msg.PName)
	fields = append(fields, fmt.Sprintf("%d", msg.PID))
	fields = append(fields, fmt.Sprintf("%d", msg.Level))
	fields = append(fields, msg.ErrMsg)

	c := s.remoteConfig()
	ret := attaAPI.SendFields(c.attaInfo.BizAttaID, c.attaInfo.BizAttaToken, fields, true)
	if ret != 0 {
		log.Printf("trpc_report_api_go: ReportBusiness atta send fail, ret:%d", ret)
		return ErrorAttaSend
	}
	return nil
}
