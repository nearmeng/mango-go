package pcgmonitor

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

const (
	// 日志自监控上报监控想名称
	hawkLogReportMonitor = "tm_hawk_log_api_report"
	// 日志自监控上报维度个数
	hawkLogMonitorDimensionCount = 8
	//  日志自监控上报指标个数
	hawkLogMonitorValuesCount = 3
)

// getConfigMonitor 中控自监控
type getConfigMonitor struct {
	configVersion int
	code          int
	count         int
	costMs        int
	succCount     int
	failCount     int
}

// sendAttaMonitor 写atta自监控
type sendAttaMonitor struct {
	ret              int
	count            int
	costMs           int
	statContentCount int
	attaID           string
}

// sendReportMonitor 业务上报自监控
type sendReportMonitor struct {
	count         int
	failCount     int
	reportType    reportType
	logsCount     int
	flowLossCount int
}

// sendLogMonitor 日志上报自监控
type sendHawkLogMonitor struct {
	count     int
	failCount int
	logName   string
	logLevel  string
	logSize   int
}

// selfMonitorLooper 自监控,周期上报，自监控数据参考 https://iwiki.woa.com/pages/viewpage.action?pageId=98470173
type selfMonitorLooper struct {
	configMutex           sync.RWMutex
	configMonitors        map[string]*getConfigMonitor // key:configVersion&code
	attaMutex             sync.RWMutex
	attaMonitors          map[string]*sendAttaMonitor // key:attaID+ret
	hawkLogAttaMutex      sync.RWMutex
	hawkLogAttaMonitors   map[string]*sendAttaMonitor // key:attaID+ret+logName
	reportMutex           sync.RWMutex
	reportMonitors        map[string]*sendReportMonitor // key:logName
	hawkLogReportMutex    sync.RWMutex
	hawkLogReportMonitors map[string]*sendHawkLogMonitor // key:logName

	active chan int
	inst   *Instance
}

// Action 具体动作,上报自监控数据
func (l *selfMonitorLooper) Action() error {
	statLogs := l.aggregation()
	return l.inst.sendSelfMonitorToAtta(statLogs)
}

// aggregation 自监控数据汇聚
func (l *selfMonitorLooper) aggregation() []*nmnt.StatLog {
	size := len(l.configMonitors) + len(l.reportMonitors) + len(l.attaMonitors)
	statLogs := make([]*nmnt.StatLog, 0, size)

	// 中控自监控
	var masterTotalCnt, masterFailCnt int
	l.configMutex.Lock()
	for k, configMonitor := range l.configMonitors {
		statLog := l.getConfigMonitorStatLog(configMonitor)
		masterTotalCnt += configMonitor.count
		masterFailCnt += configMonitor.failCount
		delete(l.configMonitors, k)
		statLogs = append(statLogs, statLog)
	}
	l.configMutex.Unlock()
	l.inst.reportLog(levelDebug, false, "Master monitor: total_num:%d, "+
		"failed_num:%d", masterTotalCnt, masterFailCnt)

	// 业务上报自监控
	l.reportMutex.Lock()
	for logName, reportMonitor := range l.reportMonitors {
		statLog := l.getReportMonitorStatLog(logName, reportMonitor)
		l.inst.reportLog(levelDebug, false, "Report monitor:%s, total_num:%d, failed_num:%d, "+
			"convergence_num:%d， flow_loss_sum:%d", logName, reportMonitor.count, reportMonitor.failCount,
			reportMonitor.logsCount, reportMonitor.flowLossCount)
		delete(l.reportMonitors, logName)
		statLogs = append(statLogs, statLog)
	}
	l.reportMutex.Unlock()

	// atta自监控
	l.attaMutex.Lock()
	for k, attaMonitor := range l.attaMonitors {
		statLog := l.getAttaMonitorStatlog(attaMonitor)
		l.inst.reportLog(levelDebug, false, "Atta monitor:%s, total_num:%d, statContentNum:%d",
			k, attaMonitor.count, attaMonitor.statContentCount)
		delete(l.attaMonitors, k)
		statLogs = append(statLogs, statLog)
	}
	l.attaMutex.Unlock()

	//
	l.hawkLogAttaMutex.Lock()
	for k, attaMonitor := range l.hawkLogAttaMonitors {
		statLog := l.getAttaMonitorStatlog(attaMonitor)
		statLog.Content[0].Dimensions = append(statLog.Content[0].Dimensions, hawklog)
		l.inst.reportLog(levelDebug, false, "Atta monitor:%s, total_num:%d, statContentNum:%d",
			k, attaMonitor.count, attaMonitor.statContentCount)

		delete(l.hawkLogAttaMonitors, k)
		statLogs = append(statLogs, statLog)
	}
	l.hawkLogAttaMutex.Unlock()

	// 日志自监控
	l.hawkLogReportMutex.Lock()
	for k, hawkLogMonitor := range l.hawkLogReportMonitors {
		statLog := l.getHawkLogMonitorStatLog(hawkLogMonitor)
		l.inst.reportLog(levelDebug, false, "LOG monitor:%s, total_num:%d, failed_num:%d",
			k, hawkLogMonitor.count, hawkLogMonitor.failCount)
		delete(l.hawkLogReportMonitors, k)
		statLogs = append(statLogs, statLog)
	}
	l.hawkLogReportMutex.Unlock()
	return statLogs
}

// Interval 时间间隔
func (l *selfMonitorLooper) Interval() time.Duration {
	interval := 1 * time.Minute
	c := l.inst.remoteConfig()

	// 根据http返回的interval来设置下一次发送的间隔时间
	if c.attaInfo.TmInterval >= 1*60*1000 {
		interval = time.Millisecond * time.Duration(c.attaInfo.TmInterval)
	}
	return interval
}

// Trigger 外部是否主动促发
func (l *selfMonitorLooper) Trigger() chan int {
	return l.active
}

// addGetConfigSelfMonitor 拉取配置的自监控上报 configVersion: 配置版本 code: 业务逻辑返回码 timeMs: http接口耗时
func (l *selfMonitorLooper) addGetConfigSelfMonitor(configVersion, code, timeMs *int) {
	l.configMutex.Lock()
	defer l.configMutex.Unlock()

	key := fmt.Sprintf("%d_%d", *configVersion, *code)
	configMonitor, ok := l.configMonitors[key]
	if ok {
		updateConfigMonitor(configMonitor, *code, *timeMs)
		return
	}

	t := &getConfigMonitor{
		configVersion: *configVersion,
		code:          *code,
		count:         0,
		costMs:        0,
		succCount:     0,
		failCount:     0,
	}
	updateConfigMonitor(t, *code, *timeMs)
	l.configMonitors[key] = t
}

// updateConfigMonitor 更新ConfigMonitor
func updateConfigMonitor(configMonitor *getConfigMonitor, code, timeMs int) {
	configMonitor.count++
	if code == 0 || code == 1 {
		configMonitor.costMs = timeMs
		configMonitor.succCount++
	} else {
		configMonitor.failCount++
	}
}

// addSendAttaSelfMonitor 发送atta自监控上报 ret: atta返回码 timeMs: 上报耗时 count: 批量上报的条数
func (l *selfMonitorLooper) addSendAttaSelfMonitor(ret, timeMs, count int, attaID string) {
	l.attaMutex.Lock()
	defer l.attaMutex.Unlock()

	key := fmt.Sprintf("%s_%d", attaID, ret)
	attaMonitor, ok := l.attaMonitors[key]
	if ok {
		attaMonitor.count++
		attaMonitor.costMs += timeMs
		attaMonitor.statContentCount += count
		return
	}

	l.attaMonitors[key] = &sendAttaMonitor{
		count:            1,
		costMs:           timeMs,
		statContentCount: count,
		attaID:           attaID,
	}
}

// addSendAttaSelfMonitor 发送atta自监控上报 ret: atta返回码 timeMs: 上报耗时 count: 批量上报的条数
func (l *selfMonitorLooper) addLogSendAttaSelfMonitor(ret, timeMs, count int, attaID, logName string) {
	l.attaMutex.Lock()
	defer l.attaMutex.Unlock()

	key := fmt.Sprintf("%s_%s_%d", logName, attaID, ret)
	attaMonitor, ok := l.hawkLogAttaMonitors[key]
	if ok {
		attaMonitor.count++
		attaMonitor.costMs += timeMs
		attaMonitor.statContentCount += count
		return
	}

	l.hawkLogAttaMonitors[key] = &sendAttaMonitor{
		count:            1,
		costMs:           timeMs,
		statContentCount: count,
		attaID:           attaID,
	}
}

// addSendReportSelfMonitor 上报自监控 ret 0-成功 1-失败 reportType 上报类型
func (l *selfMonitorLooper) addSendReportSelfMonitor(ret int, reportType reportType, logName string) {
	l.reportMutex.Lock()
	defer l.reportMutex.Unlock()

	reportMonitor, ok := l.reportMonitors[logName]
	if ok {
		updateReportMonitor(reportMonitor, ret, reportType)
		return
	}

	t := &sendReportMonitor{
		count:         0,
		failCount:     0,
		reportType:    0,
		logsCount:     0,
		flowLossCount: 0,
	}
	updateReportMonitor(t, ret, reportType)
	l.reportMonitors[logName] = t
}

// addSendHawkLogSelfMonitor 上报007日志自监控 ret 0-成功 1-失败 logName 日志名称 日志级别
func (l *selfMonitorLooper) addSendHawkLogSelfMonitor(ret, logSize int, logName, logLevel string) {
	l.hawkLogReportMutex.Lock()
	defer l.hawkLogReportMutex.Unlock()
	logKey := fmt.Sprintf("%s_%s", logName, logLevel)
	reportMonitor, ok := l.hawkLogReportMonitors[logKey]
	if ok {
		updateHawkLogMonitor(ret, reportMonitor)
		reportMonitor.logSize += logSize
		return
	}

	t := &sendHawkLogMonitor{
		count:     0,
		failCount: 0,
		logName:   logName,
		logLevel:  logLevel,
		logSize:   logSize,
	}
	updateHawkLogMonitor(ret, t)
	l.hawkLogReportMonitors[logKey] = t
}

func updateHawkLogMonitor(ret int, hawkLogMonitor *sendHawkLogMonitor) {
	hawkLogMonitor.count++
	if ret != 0 {
		hawkLogMonitor.failCount++
	}
}

// addSendReportSelfMonitorLogsCount 累加log数量
func (l *selfMonitorLooper) addSendReportSelfMonitorLogsCount(logName string) {
	l.reportMutex.Lock()
	defer l.reportMutex.Unlock()

	reportMonitor, ok := l.reportMonitors[logName]
	if ok {
		reportMonitor.logsCount++
		return
	}

	l.reportMonitors[logName] = &sendReportMonitor{
		count:         0,
		failCount:     0,
		reportType:    0,
		logsCount:     1,
		flowLossCount: 0,
	}
}

// addSendReportSelfMonitorFlowLossCount 流控丢失条数
func (l *selfMonitorLooper) addSendReportSelfMonitorFlowLossCount(logName string) {
	l.reportMutex.Lock()
	defer l.reportMutex.Unlock()

	reportMonitor, ok := l.reportMonitors[logName]
	if ok {
		reportMonitor.flowLossCount += 1
		return
	}

	l.reportMonitors[logName] = &sendReportMonitor{
		count:         0,
		failCount:     0,
		reportType:    0,
		logsCount:     0,
		flowLossCount: 1,
	}
}

// updateReportMonitorCount 更新ReportMonitor
func updateReportMonitor(reportMonitor *sendReportMonitor, ret int, reportType reportType) {
	reportMonitor.reportType = reportType
	reportMonitor.count++
	if ret != 0 {
		reportMonitor.failCount++
	}
}

// getConfigMonitorStatLog 中控汇聚数据 结构转换成上报数据
func (l *selfMonitorLooper) getConfigMonitorStatLog(configMonitor *getConfigMonitor) *nmnt.StatLog {
	statContent := &nmnt.StatContent{
		Dimensions: make([]string, 0, 8),
		Values:     make([]*nmnt.StatValue, 0, 4),
		Time:       time.Now().UnixNano() / 1e6,
	}

	l.staticDimensions(statContent)
	statContent.Dimensions = append(statContent.Dimensions, strconv.Itoa(configMonitor.configVersion))
	statContent.Dimensions = append(statContent.Dimensions, strconv.Itoa(configMonitor.code))
	if l.inst.svrInfoType == svrInfoFrame {
		statContent.Dimensions = append(statContent.Dimensions, l.inst.frameSvrInfo.FrameCode)
	} else {
		statContent.Dimensions = append(statContent.Dimensions, commReportFrameCode)
	}

	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(configMonitor.count), Count: int32(configMonitor.count),
			Policy: nmnt.Policy_SUM}) // 总请求量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(configMonitor.costMs), Count: int32(configMonitor.count),
			Policy: nmnt.Policy_SUM}) // 总耗时
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(configMonitor.succCount), Count: int32(configMonitor.count),
			Policy: nmnt.Policy_SUM}) // 成功量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(configMonitor.failCount), Count: int32(configMonitor.count),
			Policy: nmnt.Policy_SUM}) // 失败量

	statLog := &nmnt.StatLog{
		Logname: l.inst.remoteConfig().attaInfo.TmMasterName,
		Content: []*nmnt.StatContent{statContent},
	}
	return statLog
}

// getReportMonitorStatLog 上报汇聚数据 结构转成上报数据
func (l *selfMonitorLooper) getReportMonitorStatLog(logName string, reportMonitor *sendReportMonitor) *nmnt.StatLog {
	statContent := &nmnt.StatContent{
		Dimensions: make([]string, 0, 7),
		Values:     make([]*nmnt.StatValue, 0, 2),
		Time:       time.Now().UnixNano() / 1e6,
	}

	l.staticDimensions(statContent)
	statContent.Dimensions = append(statContent.Dimensions, strconv.Itoa(int(reportMonitor.reportType))) // 接口类型
	if l.inst.svrInfoType == svrInfoFrame {
		statContent.Dimensions = append(statContent.Dimensions, l.inst.frameSvrInfo.FrameCode)
	} else {
		statContent.Dimensions = append(statContent.Dimensions, commReportFrameCode)
	}
	statContent.Dimensions = append(statContent.Dimensions, logName)

	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(reportMonitor.count), Count: int32(reportMonitor.count),
			Policy: nmnt.Policy_SUM}) // 总调用量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(reportMonitor.failCount), Count: int32(reportMonitor.count),
			Policy: nmnt.Policy_SUM}) // 失败调用量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(reportMonitor.logsCount), Count: int32(reportMonitor.logsCount),
			Policy: nmnt.Policy_SUM}) // 汇聚后的上报量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(reportMonitor.flowLossCount), Count: int32(reportMonitor.flowLossCount),
			Policy: nmnt.Policy_SUM}) // 流控丢失的汇聚条数

	statLog := &nmnt.StatLog{
		Logname: l.inst.remoteConfig().attaInfo.TmReportName,
		Content: []*nmnt.StatContent{statContent},
	}
	return statLog
}

// getAttaMonitorStatlog atta发送汇聚数据 结构转成上报数据
func (l *selfMonitorLooper) getAttaMonitorStatlog(attaMonitor *sendAttaMonitor) *nmnt.StatLog {
	statContent := &nmnt.StatContent{
		Dimensions: make([]string, 0, 7),
		Values:     make([]*nmnt.StatValue, 0, 3),
		Time:       time.Now().UnixNano() / 1e6,
	}

	l.staticDimensions(statContent)
	statContent.Dimensions = append(statContent.Dimensions, strconv.Itoa(attaMonitor.ret))
	if l.inst.svrInfoType == svrInfoFrame {
		statContent.Dimensions = append(statContent.Dimensions, l.inst.frameSvrInfo.FrameCode)
	} else {
		statContent.Dimensions = append(statContent.Dimensions, commReportFrameCode)
	}
	statContent.Dimensions = append(statContent.Dimensions, attaMonitor.attaID)

	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(attaMonitor.count), Count: int32(attaMonitor.count),
			Policy: nmnt.Policy_SUM}) // 发送量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(attaMonitor.costMs), Count: int32(attaMonitor.count),
			Policy: nmnt.Policy_SUM}) // 总耗时
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(attaMonitor.statContentCount), Count: int32(attaMonitor.count),
			Policy: nmnt.Policy_SUM}) // StatContent条数

	statLog := &nmnt.StatLog{
		Logname: l.inst.remoteConfig().attaInfo.TmAttaName,
		Content: []*nmnt.StatContent{statContent},
	}
	return statLog
}

// getHawkLogMonitorStatLog 日志上报自监控
func (l *selfMonitorLooper) getHawkLogMonitorStatLog(logMonitor *sendHawkLogMonitor) *nmnt.StatLog {
	statContent := &nmnt.StatContent{
		Dimensions: make([]string, 0, hawkLogMonitorDimensionCount),
		Values:     make([]*nmnt.StatValue, 0, hawkLogMonitorValuesCount),
		Time:       time.Now().UnixNano() / 1e6,
	}
	l.staticDimensions(statContent)

	if l.inst.svrInfoType == svrInfoFrame {
		statContent.Dimensions = append(statContent.Dimensions, l.inst.frameSvrInfo.FrameCode)
	} else {
		statContent.Dimensions = append(statContent.Dimensions, commReportFrameCode)
	}
	statContent.Dimensions = append(statContent.Dimensions, logMonitor.logName)
	statContent.Dimensions = append(statContent.Dimensions, logMonitor.logLevel)

	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(logMonitor.count), Count: int32(logMonitor.count),
			Policy: nmnt.Policy_SUM}) // 发送量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(logMonitor.failCount), Count: int32(logMonitor.count),
			Policy: nmnt.Policy_SUM}) // 发送失败量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(logMonitor.logSize), Count: int32(logMonitor.count),
			Policy: nmnt.Policy_SUM}) // 数据量大小
	statLog := &nmnt.StatLog{
		Logname: hawkLogReportMonitor,
		Content: []*nmnt.StatContent{statContent},
	}
	return statLog
}

// staticDimensions 自监控公共维度
func (l *selfMonitorLooper) staticDimensions(statContent *nmnt.StatContent) {
	if l.inst.svrInfoType == svrInfoFrame {
		statContent.Dimensions = append(statContent.Dimensions,
			l.inst.frameSvrInfo.App+"."+l.inst.frameSvrInfo.Server) // 主调app.server
		statContent.Dimensions = append(statContent.Dimensions, l.inst.frameSvrInfo.IP)
		statContent.Dimensions = append(statContent.Dimensions, l.inst.frameSvrInfo.Container)
	} else {
		statContent.Dimensions = append(statContent.Dimensions, l.inst.commSvrInfo.CommName)
		statContent.Dimensions = append(statContent.Dimensions, l.inst.commSvrInfo.IP)
		statContent.Dimensions = append(statContent.Dimensions, l.inst.commSvrInfo.Container)
	}
	statContent.Dimensions = append(statContent.Dimensions, version())
	statContent.Dimensions = append(statContent.Dimensions, language)
}
