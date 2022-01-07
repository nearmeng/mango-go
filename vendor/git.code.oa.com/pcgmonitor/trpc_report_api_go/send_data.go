package pcgmonitor

import (
	"fmt"
	"log"
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/sample"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
	"github.com/golang/protobuf/proto"
)

// attaStatLogs StatLogs按atta实例聚合
type attaStatLogs struct {
	attaID    string
	attaToken string
	statLogs  []*nmnt.StatLog
}

// sendStatLogsToAtta 发送上报数据
func (s *Instance) sendStatLogsToAtta(data []*nmnt.StatLog) {
	if len(data) == 0 {
		return
	}

	// 分租
	c := s.remoteConfig()
	logsGroup := make(map[string]*attaStatLogs)
	for _, logs := range data {
		s.selfMonitorLooper.addSendReportSelfMonitorLogsCount(logs.Logname)

		// pass, err := s.isFlowControlPass(logs)
		// if err == nil && !pass {
		// 	s.selfMonitorLooper.addSendReportSelfMonitorFlowLossCount(logs.Logname)
		// 	continue
		// }

		attaID, attaToken := getAttaInfo(c, logs.Logname)
		if _, ok := logsGroup[attaID]; !ok {
			logsGroup[attaID] = &attaStatLogs{
				attaID:    attaID,
				attaToken: attaToken,
				statLogs:  []*nmnt.StatLog{logs},
			}
			continue
		}

		logsGroup[attaID].statLogs = append(logsGroup[attaID].statLogs, logs)
	}

	for _, data := range logsGroup {
		s.sendStatLogs(data)
	}
}

// sendStatLogs 发送监控项数据
func (s *Instance) sendStatLogs(data *attaStatLogs) {
	// 分批发送
	sendPreSize := len(data.statLogs)
	for start := 0; start < len(data.statLogs); {
		end := start + sendPreSize
		if end > len(data.statLogs) {
			end = len(data.statLogs)
		}

		statLogList := &nmnt.StatLogList{
			Log: data.statLogs[start:end],
		}

		begin := time.Now()
		attID, ret, lenExceed := s.sendStatLogsBiz(data.attaID, data.attaToken, statLogList)
		if lenExceed && sendPreSize > 10 {
			sendPreSize /= 2
			continue
		}
		timeMs := int(time.Since(begin) / time.Millisecond)
		s.selfMonitorLooper.addSendAttaSelfMonitor(ret, timeMs, len(statLogList.Log), attID) // atta发送自监控

		start = end
	}
}

// sendStatLogsBiz 发送监控项数据
func (s *Instance) sendStatLogsBiz(attaID, attaToken string, statLogList *nmnt.StatLogList) (string, int, bool) {
	c := s.remoteConfig()
	sample.Reduction(c.attaInfo.SamplingRate, statLogList)

	data, err := proto.Marshal(statLogList)
	if err != nil {
		log.Printf("trpc_report_api_go:statLogList proto marchal error:%v", err)
		return "", -1, false
	}

	if len(data) >= 60*1024 {
		return "", -1, true
	}

	if s.debugLogOpen {
		log.Printf("trpc_report_api_go:attaID:%s, attaToken:%s, statLogList:%+v\n", attaID, attaToken, statLogList)
	}
	// 调用atta接口上报数据
	ret := attaAPI.SendBinary(attaID, attaToken, data)
	if ret != 0 {
		log.Printf("trpc_report_api_go:atta send fail, ret:%d", ret)
	}
	return attaID, ret, false
}

// getAttaInfo 获取logName对应的atta实例
func getAttaInfo(c *remoteConfig, logName string) (attaID, attaToken string) {
	attaID = c.attaInfo.AttaID
	attaToken = c.attaInfo.AttaToken

	detailConfig, ok := c.details[logName]
	if !ok {
		return
	}

	if detailConfig.AttaID == "" || detailConfig.AttaToken == "" {
		return
	}

	attaID = detailConfig.AttaID
	attaToken = detailConfig.AttaToken
	return
}

// sendSelfMonitorToAtta 发送自监控数据
func (s *Instance) sendSelfMonitorToAtta(statLogs []*nmnt.StatLog) error {
	if len(statLogs) == 0 {
		return nil
	}

	statLogList := &nmnt.StatLogList{
		Log: statLogs,
	}

	c := s.remoteConfig()
	sample.Reduction(c.attaInfo.SamplingRate, statLogList)

	if s.debugLogOpen {
		log.Printf("trpc_report_api_go: self monitor attaID:%s, attaToken:%s, statLogList:%+v\n",
			c.attaInfo.TmAttaID, c.attaInfo.TmAttaToken, statLogList)
	}

	data, err := proto.Marshal(statLogList)
	if err != nil {
		log.Printf("trpc_report_api_go:self monitor statLogList proto marchal error:%v", err)
		return err
	}

	// 调用atta接口上报数据
	ret := attaAPI.SendBinary(c.attaInfo.TmAttaID, c.attaInfo.TmAttaToken, data)
	if ret != 0 {
		log.Printf("trpc_report_api_go:self monitor atta send fail, ret:%d", ret)
		return fmt.Errorf("atta send fail ret:%d", ret)
	}
	return nil
}
