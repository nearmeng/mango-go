package pcgmonitor

import (
	"time"

	fc "git.code.oa.com/pcgmonitor/trpc_report_api_go/api/flow_control"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

// flowControlLooper 配额数据定时整理
type flowControlLooper struct {
	active chan int
	inst   *Instance
}

// Action 循环具体动作
func (l *flowControlLooper) Action() error {
	l.inst.flowDatas.Range(func(key, value interface{}) bool {
		logName, _ := key.(string)
		f, _ := value.(*flow)
		l.reorganize(logName, f)
		return true
	})
	return nil
}

func (l *flowControlLooper) reorganize(logName string, flow *flow) error {
	now := time.Now().Unix()
	remain := make([]*fc.Unit, 0, 4)
	for _, d := range flow.Datas {
		if d.Timestamp > now {
			remain = append(remain, d)
		}
	}

	if len(remain) > 1 {
		flow.Lock()
		flow.Datas = remain
		flow.IsReport = false
		flow.Unlock()
		return nil
	}

	if !flow.IsReport {
		l.inst.flowDatas.Delete(logName)
		return nil
	}

	fcReq := &fc.ReqInfo{
		LogName:   logName,
		Status:    0,
		Quote:     0,
		TimeStamp: now,
		NodeID:    l.inst.uniqueID,
	}
	fcResp, err := fc.GetFlowControl(l.inst.polaris, fcReq)
	if err != nil {
		return err
	}

	remain = append(remain, fcResp.Data...)
	flow.Lock()
	flow.Datas = remain
	flow.IsReport = false
	flow.Unlock()
	return nil
}

// Interval 循环间隔
func (l *flowControlLooper) Interval() time.Duration {
	return 30 * time.Second
}

// Trigger 外部是否主动促发
func (l *flowControlLooper) Trigger() chan int {
	return l.active
}

func (s *Instance) isFlowControlPass(statLog *nmnt.StatLog) (bool, error) {
	f := &flow{}
	v, ok := s.flowDatas.Load(statLog.Logname)
	if ok {
		f, _ = v.(*flow)
	} else {
		s.flowDatas.Store(statLog.Logname, f)
	}

	cnt := int64(len(statLog.Content))
	unit, err := s.getUnit(f, statLog)
	// log.Printf("flowControl logName:%s, unit:%+v, statLogCnt:%d", statLog.Logname, unit, cnt)
	if err != nil {
		return false, err
	}
	if unit == nil {
		return false, nil
	}

	if unit.Quote < cnt {
		return false, nil
	}

	f.Lock()
	unit.Quote -= cnt
	f.IsReport = true
	f.Unlock()
	return true, nil
}

func (s *Instance) getUnit(f *flow, statLog *nmnt.StatLog) (*fc.Unit, error) {
	now := time.Now().Unix()
	var unit *fc.Unit
	f.Lock()
	for _, d := range f.Datas {
		if d.Timestamp > now {
			unit = d
			break
		}
	}
	f.Unlock()

	if unit == nil {
		fcReq := &fc.ReqInfo{
			LogName:   statLog.Logname,
			Status:    0,
			Quote:     0,
			TimeStamp: now,
			NodeID:    s.uniqueID,
		}
		fcRsp, err := fc.GetFlowControl(s.polaris, fcReq)
		if err != nil {
			return nil, err
		}

		f.Lock()
		f.Datas = fcRsp.Data
		for _, d := range f.Datas {
			if d.Timestamp > now {
				unit = d
				break
			}
		}
		f.Unlock()
	}

	if unit == nil {
		return nil, nil
	}

	cnt := int64(len(statLog.Content))

	if unit.Quote < cnt {
		fcReq := &fc.ReqInfo{
			LogName:   statLog.Logname,
			Status:    1,
			Quote:     cnt - unit.Quote,
			TimeStamp: now,
			NodeID:    s.uniqueID,
		}
		fcRsp, err := fc.GetFlowControl(s.polaris, fcReq)
		if err != nil {
			if err == fc.ErrorFCOverflow {
				return nil, nil
			}
			return nil, err
		}

		f.Lock()
		unit.Quote += fcRsp.Data[0].Quote
		f.Unlock()
	}
	return unit, nil
}
