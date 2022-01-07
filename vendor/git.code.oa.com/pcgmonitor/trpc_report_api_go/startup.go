package pcgmonitor

import (
	"strings"
	"sync"
	"time"

	"git.code.oa.com/atta/attaapi-go"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/config"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/loop"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/route"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
	"github.com/google/uuid"
)

// Setup [框架]初始化
func (s *Instance) Setup(info *FrameSvrSetupInfo) error {
	if s.svrInfoType != svrInfoUnknown {
		return ErrorInitRepeat
	}

	if info == nil {
		return ErrorInitParamInvalid
	}

	s.svrInfoType = svrInfoFrame

	s.frameSvrInfo.App = info.App
	s.frameSvrInfo.Server = info.Server
	s.frameSvrInfo.IP = info.IP
	s.frameSvrInfo.Container = info.Container
	s.frameSvrInfo.ConSetId = info.ConSetId
	s.frameSvrInfo.Version = info.Version
	s.frameSvrInfo.PhysicEnv = info.PhysicEnv
	s.frameSvrInfo.UserEnv = info.UserEnv
	s.frameSvrInfo.FrameCode = info.FrameCode
	s.frameSvrInfo.DebugLogOpen = info.DebugLogOpen

	s.hawkLogNames = info.HawkLogNames

	s.polarisInfo.Addrs = info.Addrs
	s.polarisInfo.Proto = info.Proto

	s.debugLogOpen = info.DebugLogOpen

	s.uniqueID = uuid.New().String()

	return s.startup()
}

// CommSetUp [非框架]初始化
func (s *Instance) CommSetUp(info *CommSvrSetupInfo) error {
	if s.svrInfoType != svrInfoUnknown {
		return ErrorInitRepeat
	}

	if info == nil {
		return ErrorInitParamInvalid
	}

	s.svrInfoType = svrInfoComm

	s.commSvrInfo.CommName = info.CommName
	s.commSvrInfo.IP = info.IP
	s.commSvrInfo.Container = info.Container
	s.commSvrInfo.DebugLogOpen = info.DebugLogOpen

	s.polarisInfo.Addrs = info.Addrs
	s.polarisInfo.Proto = info.Proto

	s.debugLogOpen = info.DebugLogOpen

	s.uniqueID = uuid.New().String()

	s.hawkLogNames = info.HawkLogNames

	return s.startup()
}

// NewInstance New007上报实例
func NewInstance() *Instance {
	c := config.DefaultData()
	rc := &remoteConfig{
		configVersion: 0,
		prefixInfo: &prefixInfo{
			prefix:               c.Prefix,
			prefixMetrics:        c.Prefix + "m",
			prefixActiveModCall:  c.Prefix + "a",
			prefixPassiveModCall: c.Prefix + "p",
			prefixCustom:         c.Prefix + "c",
		},
		attaInfo: c,
	}

	s := new(Instance)
	s.withRemoteConfig(rc)
	s.withHawkLogConfig(&hawkLogConfig{})

	s.svrInfoType = svrInfoUnknown
	s.frameSvrInfo = &FrameSvrInfo{}
	s.commSvrInfo = &CommSvrInfo{}
	s.polarisInfo = &PolarisInfo{}
	s.polaris = &route.Polaris{}

	// 管道初始化
	s.channel = make(chan *nmnt.StatLog, channelMaxSize)
	// 007远程配置 周期拉取
	s.configLooper = &configLooper{
		active: make(chan int, 1), inst: s,
	}
	// 数据周期上报
	s.reportLooper = &reportLooper{
		active: make(chan int, 1), inst: s,
	}
	// 自监控周期上报
	s.selfMonitorLooper = &selfMonitorLooper{
		configMutex:           sync.RWMutex{},
		configMonitors:        make(map[string]*getConfigMonitor),
		attaMutex:             sync.RWMutex{},
		attaMonitors:          make(map[string]*sendAttaMonitor),
		hawkLogAttaMutex:      sync.RWMutex{},
		hawkLogAttaMonitors:   make(map[string]*sendAttaMonitor),
		reportMutex:           sync.RWMutex{},
		reportMonitors:        make(map[string]*sendReportMonitor),
		hawkLogReportMutex:    sync.RWMutex{},
		hawkLogReportMonitors: make(map[string]*sendHawkLogMonitor),
		active:                make(chan int, 1),
		inst:                  s,
	}
	s.flowControlLooper = &flowControlLooper{
		active: make(chan int, 1),
		inst:   s,
	}
	s.logChannel = make(chan *logChanData, logChannelMaxSize)
	s.logReportLooper = &logReportLooper{
		active: make(chan int, 1),
		inst:   s,
	}
	logger := new(m007Logger)
	logger.Inst = s
	s.logger = logger
	return s
}

// startup 启动逻辑
func (s *Instance) startup() error {
	var err error

	start := time.Now()
	// atta初始化
	attaInitOnce.Do(func() { s.attaStartup(&err) })
	if err != nil {
		return err
	}
	s.reportLog(levelDebug, s.debugLogOpen, "trpc_report_api_go:atta init, total costMs[%d]",
		time.Since(start)/time.Millisecond)

	// 北极星初始化
	adds := strings.Split(s.polarisInfo.Addrs, ",")
	err = s.polaris.Init(adds, s.polarisInfo.Proto)
	if err != nil {
		return err
	}
	s.reportLog(levelDebug, s.debugLogOpen, "trpc_report_api_go:polaris init, total costMs[%d]",
		time.Since(start)/time.Millisecond)

	// 拉取远端配置，有兜底配置，忽略失败
	_ = s.configLooper.updateConfig(&config.ReqBody{
		ConfigInfoLogBody: &config.HawkLogBody{
			Params: []*config.HawkLogSdk{},
		},
	})

	s.reportLog(levelDebug, s.debugLogOpen, "trpc_report_api_go:get remote config, total costMs[%d]",
		time.Since(start)/time.Millisecond)

	c := s.remoteConfig()
	// 启动一个goroutine定时拉取配置
	go loop.Start(s.configLooper, time.Duration(c.attaInfo.GetConfigInterval)*time.Millisecond)
	// 启动一个goroutine去循环定时上报数据到atta
	go loop.Start(s.reportLooper, time.Duration(c.attaInfo.SendInterval)*time.Millisecond)
	// 启动一个goroutine去循环定时上报自监控数据
	go loop.Start(s.selfMonitorLooper, time.Duration(c.attaInfo.TmInterval)*time.Millisecond)
	// 启动一个goroutine去循环定时上报日志数据到atta
	go loop.Start(s.logReportLooper, logSendIntervalMilliseconds*time.Millisecond)
	// go loop.Start(s.flowControlLooper, 30*time.Second)
	s.reportLog(levelDebug, s.debugLogOpen, "trpc_report_api_go:init success, total costMs[%d]",
		time.Since(start)/time.Millisecond)

	return nil
}

// attaStartup atta初始化
func (s *Instance) attaStartup(err *error) {
	ret := attaAPI.InitUDP()
	if ret == 0 {
		return
	}

	switch ret {
	case attaapi.AttaReportCodeNoUsablePort:
		s.reportLog(levelError, true, "trpc_report_api_go:atta initUDP err, ret[%d], "+
			"Can not find AttaAgent port", ret)
		*err = ErrorInit
	case attaapi.AttaReportCodeCreateSocketFailed:
		s.reportLog(levelError, true, "trpc_report_api_go:atta initUDP err, ret[%d], "+
			"Can not create socket with AttaAgent port", ret)
	case attaapi.AttaReportCodeConnetSocketFailed:
		s.reportLog(levelError, true, "trpc_report_api_go:atta initUDP err, ret[%d], "+
			"Can not connect socket to AttaAgent port", ret)
	default:
		s.reportLog(levelError, true, "trpc_report_api_go:atta initUDP err, ret[%d]", ret)
		*err = ErrorInit
	}
}
