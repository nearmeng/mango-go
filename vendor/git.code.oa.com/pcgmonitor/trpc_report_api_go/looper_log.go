package pcgmonitor

import (
	"time"
)

// loglogReportLooper 日志周期上报
type logReportLooper struct {
	active chan int
	inst   *Instance
}

// Action 具体动作
func (l *logReportLooper) Action() error {
	l.aggregation()
	return nil
}

// Interval 循环间隔
func (l *logReportLooper) Interval() time.Duration {
	return logSendIntervalMilliseconds * time.Millisecond
}

// Trigger 外部是否主动促发
func (l *logReportLooper) Trigger() chan int {
	return l.active
}

// aggregation 日志发送处理
func (l *logReportLooper) aggregation() {
	logData := make([]*logChanData, 0, reportAttaMaxSize)
	finish := false
	length := 0
	for !finish {
		var statLog *logChanData
		statLog, finish = l.getChanResult()
		dataSize := l.getDataSize(statLog)
		// 超过数据量长度或者超过atta批量发送的条数以及channel阻塞但是有值的情况都得发送
		if l.isSendLog(length, dataSize, len(logData), finish) {
			l.inst.sendLogToAtta(logData)
			logData = logData[0:0]
			length = 0
		}
		if finish {
			break
		}
		if statLog != nil {
			length += dataSize
			logData = append(logData, statLog)
		}
	}
}

// getChanResult 从chan 中获取日志信息
func (l *logReportLooper) getChanResult() (*logChanData, bool) {
	finish := false
	var statLog *logChanData
	select {
	case statLog = <-l.inst.logChannel:
	default:
		finish = true
	}
	return statLog, finish
}

// getDataSize 获取当前日志的大小
func (l *logReportLooper) getDataSize(statLog *logChanData) int {
	dataSize := 0
	if statLog != nil {
		dataSize = len(statLog.data)
	}
	return dataSize
}

// isSendLog 是否需要发送日志
func (l *logReportLooper) isSendLog(allSize, currentSize, dataMapSize int, finish bool) bool {
	// 是否超过atta上报单条数据最大值
	var isExceedMaxSize = (allSize + currentSize) > reportDataMaxSize
	// 是否超过atta批量上报最大限制条数
	var isExceedMaxBatchSize = dataMapSize >= reportAttaMaxSize
	// channel阻塞但是有值的情况都得发送
	var isFinishSend = finish && allSize > 0
	return isExceedMaxSize || isExceedMaxBatchSize || isFinishSend
}
