package pcgmonitor

import (
	"log"
	"time"
)

// sendLogToAtta 发送日志数据
func (s *Instance) sendLogToAtta(data []*logChanData) {
	if len(data) == 0 {
		return
	}
	// 先用logName分组 然后用attaID分驻批量发送
	logsGroup := make(map[string]map[string]*logChanData)
	for _, logData := range data {
		dataMap, ok := logsGroup[logData.logName]
		if !ok {
			logsGroup[logData.logName] = map[string]*logChanData{
				logData.attaID: {
					attaID:    logData.attaID,
					attaToken: logData.attaToken,
					dataList:  [][]byte{logData.data},
				},
			}
			continue
		}
		if _, ok := dataMap[logData.attaID]; !ok {
			dataMap[logData.attaID] = &logChanData{
				attaID:    logData.attaID,
				attaToken: logData.attaToken,
				dataList:  [][]byte{logData.data},
			}
			continue
		}
		dataMap[logData.attaID].dataList = append(dataMap[logData.attaID].dataList, logData.data)
	}

	for name, dataMap := range logsGroup {
		for _, data := range dataMap {
			s.sendLogs(name, data)
		}
	}
}

// sendLogs 发送日志数据 如果日志量过大 超过atta一次发送的范围，则减半分批发送
func (s *Instance) sendLogs(logName string, data *logChanData) {
	begin := time.Now()
	// 分批发送
	dataCount := len(data.dataList)
	ret, lenExceed := s.sendLogsBiz(data.attaID, data.attaToken, data.dataList)
	if lenExceed {
		s.sendLogsBiz(data.attaID, data.attaToken, data.dataList[0:dataCount/2])
		s.sendLogsBiz(data.attaID, data.attaToken, data.dataList[dataCount/2:])
	}
	timeMs := time.Since(begin).Milliseconds()
	s.selfMonitorLooper.addLogSendAttaSelfMonitor(ret, int(timeMs), dataCount, data.attaID, logName)
}

// sendLogsBiz 发送日志数据
func (s *Instance) sendLogsBiz(attaID, attaToken string, dataList [][]byte) (int, bool) {
	if len(dataList) >= reportDataMaxSize {
		return -1, true
	}
	if s.debugLogOpen {
		log.Printf("trpc_report_api_go:attaID:%s, attaToken:%s, dataList:%s\n", attaID, attaToken, dataList)
	}
	// 调用atta接口上报数据
	ret := attaAPI.BatchSendBinary(attaID, attaToken, dataList)
	if ret != 0 {
		log.Printf("trpc_report_api_go:log send fail, ret:%d, logData:%s", ret, dataList)
	}
	return ret, false
}
