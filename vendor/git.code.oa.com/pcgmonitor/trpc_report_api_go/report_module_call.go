package pcgmonitor

import (
	"errors"
	"log"
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/config"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/sample"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/stat"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

// ActiveMsg 主调上报，保持兼容，不调整字段命名
type ActiveMsg struct {
	AService   string  // 主调Service
	AInterface string  // 主调接口
	PApp       string  // 被调APP
	PServer    string  // 被调server
	PService   string  // 被调service
	PInterface string  // 被调interface
	PIp        string  // 被调IP
	PContainer string  // 被调容器名
	PConSetId  string  // 被调容器的SetID
	PTarget    string  // 被调地址
	RetCode    string  // 业务返回码
	Status     int64   // 状态(0:成功; 1:异常; 2:超时)
	Time       float64 // 耗时，毫秒
	Etx        string  // 自定义维度
}

// PassiveMsg 被调上报，保持兼容，不调整字段命名
type PassiveMsg struct {
	AApp       string  // 主调APP
	AServer    string  // 主调server
	AService   string  // 主调service
	AInterface string  // 主调interface
	AIp        string  // 主调IP
	AContainer string  // 主调容器名
	AConSetId  string  // 主调容器的SetID
	AAddress   string  // 主调地址
	PService   string  // 被调service
	PInterface string  // 被调interface
	PIp        string  // 被调IP
	RetCode    string  // 业务返回码
	Status     int64   // 状态(0:成功; 1:异常; 2:超时)
	Time       float64 // 耗时，毫秒
	Etx        string  // 自定义维度
}

// ReportActive [框架]主调上报
func (s *Instance) ReportActive(activeMsg *ActiveMsg) error {
	if s.debugLogOpen {
		log.Printf("trpc_report_api_go:ReportActive activeMsg:%+v", activeMsg)
	}

	if s.svrInfoType != svrInfoFrame {
		return ErrorReportNotInit
	}

	c := s.remoteConfig()
	if !sample.Pass(c.attaInfo.SamplingRate) {
		return nil
	}

	statContent := &nmnt.StatContent{
		Dimensions: make([]string, 0, moduleDimensionsSize),
		Values:     make([]*nmnt.StatValue, 0, moduleValuesSize),
		Time:       time.Now().UnixNano() / 1e6,
	}

	statContent.Dimensions = append(statContent.Dimensions,
		s.frameSvrInfo.App+"."+s.frameSvrInfo.Server) // 主调app.server
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.AService)   // 主调Service
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.AInterface) // 主调接口
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.IP)
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.Container)
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.ConSetId)
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.PApp+"."+activeMsg.PServer) // 被调App.Server
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.PService)                   // 被调Service
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.PInterface)                 // 被调Interface
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.PIp)                        // 被调IP
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.PContainer)                 // 被调容器名
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.PConSetId)                  // 被调容器的SetID
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.RetCode)                    // retCode
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.Version)
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.PhysicEnv)
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.UserEnv)
	logName := s.getLogName(c.prefixInfo.prefixActiveModCall)
	statContent.Dimensions = append(statContent.Dimensions,
		config.GetTimeInternal(logName, activeMsg.Time, c.details)) // 时间区间
	statContent.Dimensions = append(statContent.Dimensions, appendLanguage(activeMsg.PTarget))
	statContent.Dimensions = append(statContent.Dimensions, activeMsg.Etx) // 自定义维度

	stat.MultiDimensionsFix(logName, statContent.Dimensions, c.details, 0)

	return s.withStatLogData(statContent, &moduleCall{activeMsg.Status, activeMsg.Time, activeMsg.RetCode},
		logName, activeReport)
}

// ReportPassive [框架]被调上报
func (s *Instance) ReportPassive(passiveMsg *PassiveMsg) error {
	if s.debugLogOpen {
		log.Printf("trpc_report_api_go:ReportPassive passiveMsg:%+v", passiveMsg)
	}

	if s.svrInfoType != svrInfoFrame {
		return ErrorReportNotInit
	}

	c := s.remoteConfig()
	if !sample.Pass(c.attaInfo.SamplingRate) {
		return nil
	}

	statContent := &nmnt.StatContent{
		Dimensions: make([]string, 0, moduleDimensionsSize),
		Values:     make([]*nmnt.StatValue, 0, moduleValuesSize),
		Time:       time.Now().UnixNano() / 1e6,
	}

	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.AApp+"."+passiveMsg.AServer) // 主调App.Server
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.AService)                    // 主调Service
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.AInterface)                  // 主调Interface
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.AIp)                         // 主调IP
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.AContainer)                  // 主调容器名
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.AConSetId)                   // 主调容器的SetID
	statContent.Dimensions = append(statContent.Dimensions,
		s.frameSvrInfo.App+"."+s.frameSvrInfo.Server) // 被调app.server
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.PService)   // 被调Service
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.PInterface) // 被调接口
	if passiveMsg.PIp != "" {
		statContent.Dimensions = append(statContent.Dimensions, passiveMsg.PIp) // 被调IP
	} else {
		statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.IP)
	}
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.Container)
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.ConSetId)
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.RetCode) // retCode
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.Version)
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.PhysicEnv)
	statContent.Dimensions = append(statContent.Dimensions, s.frameSvrInfo.UserEnv)
	logName := s.getLogName(c.prefixInfo.prefixPassiveModCall)
	statContent.Dimensions = append(statContent.Dimensions,
		config.GetTimeInternal(logName, passiveMsg.Time, c.details)) // 时间区间
	statContent.Dimensions = append(statContent.Dimensions, appendLanguage(passiveMsg.AAddress))
	statContent.Dimensions = append(statContent.Dimensions, passiveMsg.Etx) // 自定义维度

	stat.MultiDimensionsFix(logName, statContent.Dimensions, c.details, 0)

	return s.withStatLogData(statContent, &moduleCall{passiveMsg.Status, passiveMsg.Time, passiveMsg.RetCode},
		logName, passiveReport)
}

// appendLanguage 拼接语言类型给指定的字符串
func appendLanguage(value string) string {
	if value == "" {
		return ""
	}

	return language + "[" + value + "]"
}

// moduleCall 模调
type moduleCall struct {
	status  int64
	time    float64
	retCode string
}

// withStatLogData statContent公共数据
func (s *Instance) withStatLogData(statContent *nmnt.StatContent, mod *moduleCall,
	logName string, reportType reportType) error {
	success, exception, timeout, err := modAnalyse(mod)
	if err != nil {
		return err
	}

	// 总请求量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: 1, Count: 1, Policy: nmnt.Policy_SUM})
	// 平均耗时
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: mod.time, Count: 1, Policy: nmnt.Policy_AVG})
	// 成功量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(success), Count: 1, Policy: nmnt.Policy_SUM})
	// 异常量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(exception), Count: 1, Policy: nmnt.Policy_SUM})
	// 超时量
	statContent.Values = append(statContent.Values,
		&nmnt.StatValue{Value: float64(timeout), Count: 1, Policy: nmnt.Policy_SUM})

	// 数据写管道，异步上报
	return s.withStatLogCh(logName, statContent, reportType)
}

// modAnalyse 模调分析
func modAnalyse(mod *moduleCall) (success, exception, timeout int32, err error) {
	// status(框架状态) 和 retCode(业务返回码) 一起判断上报状态
	switch mod.status {
	case 0:
		if mod.retCode == "0" {
			success++
		} else {
			exception++
		}
	case 1:
		exception++
	case 2:
		timeout++
	default:
		err = errors.New("status value error")
	}
	return
}

// sendDependOnContentNum 管道数量达到汇聚条件时促发汇聚
func (s *Instance) sendDependOnContentNum() {
	c := s.remoteConfig()
	sendNum := c.attaInfo.SendNum

	if len(s.channel) >= sendNum {
		select {
		case s.reportLooper.active <- 1:
		default:
		}
	}
}
