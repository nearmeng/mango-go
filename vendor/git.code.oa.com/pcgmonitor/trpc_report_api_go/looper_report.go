package pcgmonitor

import (
	"math"
	"strings"
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

// aggregationKey 汇聚key类型
type aggregationKey string

// reportLooper 监控项数据周期汇聚上报
type reportLooper struct {
	active chan int
	inst   *Instance
}

// Action 具体动作
func (l *reportLooper) Action() error {
	statLogMap := l.aggregation()

	statLogs := make([]*nmnt.StatLog, 0, len(statLogMap))
	for _, statLog := range statLogMap {
		statLogs = append(statLogs, statLog)
	}

	l.inst.sendStatLogsToAtta(statLogs)
	return nil
}

// Interval 循环间隔
func (l *reportLooper) Interval() time.Duration {
	// 做个保护，避免太频繁拉取
	interval := 1 * time.Second
	c := l.inst.remoteConfig()
	// 根据http返回的interval来设置下一次发送的间隔时间
	if c.attaInfo.SendInterval >= 1*1000 {
		interval = time.Millisecond * time.Duration(c.attaInfo.SendInterval)
	}
	return interval
}

// Trigger 外部是否主动促发
func (l *reportLooper) Trigger() chan int {
	return l.active
}

// aggregation 监控项数据汇聚
func (l *reportLooper) aggregation() map[aggregationKey]*nmnt.StatLog {
	statLogMap := make(map[aggregationKey]*nmnt.StatLog)

	finish := false
	for !finish {
		select {
		case statLog := <-l.inst.channel:
			l.statLogAggregation(statLog, statLogMap)

		default:
			finish = true
		}
	}

	return statLogMap
}

// statLogAggregation 单条statLog汇聚
func (l *reportLooper) statLogAggregation(statLog *nmnt.StatLog, statLogMap map[aggregationKey]*nmnt.StatLog) {
	// 管道数据不符合规范,丢弃
	if len(statLog.Content) != 1 {
		return
	}

	key := getAggregationKey(statLog.Logname, statLog.Content[0].Dimensions)
	s, ok := statLogMap[key]
	if !ok {
		statLogMap[key] = statLog
		return
	}

	if len(s.Content[0].Values) != len(statLog.Content[0].Values) {
		statLogMap[key] = statLog
		return
	}

	// 运行过程中，策略不会变化
	for i, v := range s.Content[0].Values {
		t := statLog.Content[0].Values[i]
		v.Count += t.Count
		switch v.Policy {
		case nmnt.Policy_AVG:
			fallthrough
		case nmnt.Policy_SUM:
			v.Value += t.Value
		case nmnt.Policy_MAX:
			v.Value = math.Max(v.Value, t.Value)
		case nmnt.Policy_MIN:
			v.Value = math.Min(v.Value, t.Value)
		case nmnt.Policy_SET:
			v.Value = t.Value
		default:
		}
	}
}

// getAggregationKey 汇聚key，相同维度的监控项可汇聚
func getAggregationKey(logName string, dimensions []string) aggregationKey {
	var b strings.Builder
	b.WriteString(logName)
	for _, d := range dimensions {
		b.WriteString(d)
	}
	return aggregationKey(b.String())
}
