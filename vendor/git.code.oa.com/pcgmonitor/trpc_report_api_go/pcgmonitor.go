// Package pcgmonitor 007上报SDK
package pcgmonitor

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"git.code.oa.com/atta/attaapi-go"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/config"
	fc "git.code.oa.com/pcgmonitor/trpc_report_api_go/api/flow_control"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/route"
	"git.code.oa.com/pcgmonitor/trpc_report_api_go/pb/nmnt"
)

func init() {
	// 采样率，设置随机数
	rand.Seed(time.Now().Unix())
}

var (
	ErrorInitParamInvalid = errors.New("init param invalid")
	ErrorInit             = errors.New("init error")
	ErrorInitRepeat       = errors.New("init repeated")
	ErrorReportNotInit    = errors.New("not init")
	ErrorReportLossData   = errors.New("loss data")
	ErrorAttaSend         = errors.New("atta send fail")
	ErrorLogNameRequired  = errors.New("log name is require")
	ErrorLogNameNeedInit  = errors.New("logName is new LogName, need to initialize")
	ErrorLogNotSampled    = errors.New("log not be sampled")
)

var (
	defaultInstance = NewInstance() // 默认上报实例，对外提供包函数
)

// FrameSvrInfo 框架服务
type FrameSvrInfo struct {
	App       string
	Server    string
	IP        string
	Container string // 容器名
	ConSetId  string // 容器的SetID
	Version   string
	PhysicEnv string // 物理环境
	UserEnv   string // 用户环境
	FrameCode string // 框架标识：trpc,grpc...

	HawkLogNames []string // 日志名称集合

	DebugLogOpen bool
}

// CommSvrInfo 非框架服务
type CommSvrInfo struct {
	CommName  string // 非框架监控项
	IP        string
	Container string // 容器名

	HawkLogNames []string // 日志名称集合

	DebugLogOpen bool
}

// PolarisInfo 北极星配置信息
type PolarisInfo struct {
	Addrs string
	Proto string
}

// FrameSvrSetupInfo 框架服务初始化
type FrameSvrSetupInfo struct {
	FrameSvrInfo
	PolarisInfo
}

// CommSvrSetupInfo 非框架服务初始化
type CommSvrSetupInfo struct {
	CommSvrInfo
	PolarisInfo
}

// flow 单监控项流控数据
type flow struct {
	sync.Mutex
	Datas    []*fc.Unit // 分时间片 一组额度
	IsReport bool       // 是否上报过
}

// prefixInfo 监控项前缀
type prefixInfo struct {
	prefix               string
	prefixMetrics        string
	prefixActiveModCall  string
	prefixPassiveModCall string
	prefixCustom         string
}

// remoteConfig 远程配置信息
type remoteConfig struct {
	configVersion int
	prefixInfo    *prefixInfo
	attaInfo      *config.Data
	details       map[string]*config.Detail
}

type hawkLogConfig struct {
	hawkLogConfig map[string]*config.HawkLogConfig
}

// reportType 上报类型
type reportType int

const (
	activeReport  reportType = 0 // [框架]主调上报
	passiveReport reportType = 1 // [框架]被调上报
	metricsReport reportType = 2 // [框架]属性上报
	customReport  reportType = 3 // [框架]自定义上报
	commonReport  reportType = 4 // [非框架]自定义上报
)

// svrInfoType svr类型
type svrInfoType int

const (
	svrInfoUnknown svrInfoType = 0 // 未知
	svrInfoFrame   svrInfoType = 1 // 框架上报
	svrInfoComm    svrInfoType = 2 // 非框架上报
)

var (
	// 007全策略
	attrPolicies = []nmnt.Policy{nmnt.Policy_SUM, nmnt.Policy_AVG, nmnt.Policy_MAX, nmnt.Policy_MIN, nmnt.Policy_SET}

	attaAPI      attaapi.AttaApi // atta实例，全局
	attaInitOnce sync.Once       // atta实例初始化
)

const (
	language            = "GO"
	commReportFrameCode = "comm"
	hawklog             = "Log"

	channelMaxSize       = 20000 // 管道容量
	moduleDimensionsSize = 17    // 模调维度长度
	moduleValuesSize     = 5     // 模调value长度
)

//  Instance 007上报实例
type Instance struct {
	// 初始化配置
	svrInfoType  svrInfoType   // svr类型
	frameSvrInfo *FrameSvrInfo // 框架上报svrInfo
	commSvrInfo  *CommSvrInfo  // 非框架上报svrInfo
	polarisInfo  *PolarisInfo  // 北极星配置信息
	debugLogOpen bool          // 调试日志开启

	// 流控相关
	uniqueID  string   // 唯一ID
	flowDatas sync.Map // 本地流控存储，key：logName value：flow指针

	// 远程配置
	config        atomic.Value // 007远程配置，remoteConfig 无锁化
	hawkLogConfig atomic.Value // hawk日志远程配置，hawkLogConfig 无锁化

	// 循环执行逻辑
	configLooper      *configLooper      // 007远程配置周期性拉取
	reportLooper      *reportLooper      // 数据周期汇集上报
	selfMonitorLooper *selfMonitorLooper // 自监控数据周期汇集上报
	flowControlLooper *flowControlLooper // 流控数据周期拉取
	logReportLooper   *logReportLooper   //日志周期汇集上报

	polaris *route.Polaris     // 北极星, 007拉远程配置时依赖
	channel chan *nmnt.StatLog // 管道，缓存上报信息。异步上报方式 注意主goroutinue不能立刻关闭

	// 日志相关
	logChannel       chan *logChanData
	logger           Logger   // 007日志
	hawkLogNames     []string // 存储用户初始化的名称
	newHawkLogMap    sync.Map // 存储新日志名称 用于判断日志信息是否获取到了，日志名称存在在该map里面说明正在获取日志信息
	hawkLogSampleMap sync.Map // 对于有contextId的日志进行采样 用于存放contextId 及其 expire的时间
}

// remoteConfig 获取远程配置
func (s *Instance) remoteConfig() *remoteConfig {
	return s.config.Load().(*remoteConfig)
}

// withRemoteConfig 设置远程配置
func (s *Instance) withRemoteConfig(c *remoteConfig) {
	s.config.Store(c)
}

// getHawkLOGConfig 获取日志远程配置
func (s *Instance) getHawkLogConfig() *hawkLogConfig {
	return s.hawkLogConfig.Load().(*hawkLogConfig)
}

// withHawkLogConfig 设置远程配置
func (s *Instance) withHawkLogConfig(h *hawkLogConfig) {
	s.hawkLogConfig.Store(h)
}

// GetLogger 获取007 Logger
func (s *Instance) GetLogger() Logger {
	return s.logger
}
