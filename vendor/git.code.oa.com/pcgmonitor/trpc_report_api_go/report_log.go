package pcgmonitor

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"git.code.oa.com/pcgmonitor/trpc_report_api_go/api/config"
)

// HawkLogParams 上报日志需要传递的参数
type HawkLogParams struct {
	// Name 日志名称
	Name string
	// Level 日志级别
	Level HawkLogLevel
	// ContextID 全局上下文id
	ContextID string
	// Content 日志内容
	Content string
	// 日志维度
	Dimensions []string
}

// hawkLogData 发往atta的数据结构
type hawkLogData struct {
	// Time 日志上报时间
	Time string `json:"time"`
	// Dimensions 日志维度 用户自定义
	Dimensions map[string]string `json:"dimensions"`
	// Prop 日志维度 固定
	Prop map[string]string `json:"prop"`
	// Content 日志内容
	Content string `json:"content"`
}

// logChanData 暂存在chan里面日志的信息
type logChanData struct {
	attaID    string
	attaToken string
	logName   string
	data      []byte
	dataList  [][]byte
}

type HawkLogLevel int

// 日志级别 0-trace 1-debug 2-info 3-warn 4-error
const (
	TraceLevel HawkLogLevel = 0
	DebugLevel HawkLogLevel = 1
	InfoLevel  HawkLogLevel = 2
	WarnLevel  HawkLogLevel = 3
	ErrorLevel HawkLogLevel = 4
	FatalLevel HawkLogLevel = 6
)

// LogLevel 日志级别的枚举
var hawkLogLevelMap = map[HawkLogLevel]string{
	TraceLevel: "TRACE",
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

const (
	// atta 上报单条数据最大值
	reportDataMaxSize = 60 * 1024
	// atta 批量上报最大限制条数
	reportAttaMaxSize = 100
	// 上下文id的日志采样过期时间 单位s
	contextIDSampleExpireTime = 300
	// 日志上报时间格式化 精确到毫秒
	logDateFormat = "2006-01-02 15:04:05.000"
	// 日志管道容量
	logChannelMaxSize = 10000
	// 毫秒 定时出发发送日志
	logSendIntervalMilliseconds = 5000
	// 自监控正常值
	retSuccess = 0
	// 自监控不正常值
	retFailure = 1
)

// ReportHawkLog 上报日志通用方法
func (s *Instance) ReportHawkLog(params *HawkLogParams) error {
	if params.Name == "" {
		return ErrorLogNameRequired
	}
	// 检查名称是否初始化了
	if !checkStrInSlice(s.hawkLogNames, params.Name) {
		return ErrorLogNameNeedInit
	}

	if s.svrInfoType == svrInfoFrame {
		// 框架上报添加必要的维度
		staticDimensions := []string{s.frameSvrInfo.IP, s.frameSvrInfo.Container,
			s.frameSvrInfo.ConSetId, s.frameSvrInfo.PhysicEnv, s.frameSvrInfo.UserEnv}
		params.Dimensions = append(staticDimensions, params.Dimensions...)
	}
	// 获取中控日志信息
	c := s.getHawkLogConfig()
	// 中控返回的日志信息
	hawkLogConfig, logKeyExist := c.hawkLogConfig[params.Name]
	isUpdate := checkUpdateConf(params.Name, len(params.Dimensions), c)
	// 日志名称或者当前日志的日志级别不存在  或者 当前日志的维度有增加
	if isUpdate {
		// 如果有 就说明 已经请求中控拉取信息了，不必要发送第二次请求
		if _, ok := s.newHawkLogMap.Load(params.Name); !ok {
			s.newHawkLogMap.Store(params.Name, 1)
			// 新的日志名称或者日志级别 version需要设置为0
			version := 0
			if logKeyExist {
				version = hawkLogConfig.Version
			}
			// 去中控获取更新后的日志信息
			go s.getConfigWithLog(params.Name, version, len(params.Dimensions))
		}
	}
	// 日志名称 没在中控信息里面直接返回 等待获取到中控信息才能上报
	if !logKeyExist {
		return ErrorLogNameNeedInit
	}

	// 采样
	if !s.logSend(params, hawkLogConfig.SamplingRate) {
		return ErrorLogNotSampled
	}
	// 过滤需要过滤的字段
	if len(hawkLogConfig.Ignore) > 0 {
		filterParams(hawkLogConfig.Ignore, params)
	}
	// 发送数据到atta
	s.sendHawkLogData(hawkLogConfig.AttaID, hawkLogConfig.AttaToken, params)
	separateAttaInfo, ok := hawkLogConfig.SubMaps[int(params.Level)]
	if ok {
		// 发送数据到atta
		s.sendHawkLogData(separateAttaInfo.AttaID, separateAttaInfo.AttaToken, params)
	}
	return nil
}

// checkUpdateConf 判断是否需要更新配置
func checkUpdateConf(logName string, dimensionsNum int, logConf *hawkLogConfig) bool {
	hawkLogConfig, ok := logConf.hawkLogConfig[logName]
	// 当前日志名称 或者 日志级别没有在配置中
	if !ok {
		return true
	}
	// 当前日志名称对应的当前级别的维度有增加
	if dimensionsNum > hawkLogConfig.DimensionsNum {
		return true
	}
	return false
}

// assembleLogData 组装数据
func (s *Instance) assembleLogData(params *HawkLogParams) *hawkLogData {
	logData := new(hawkLogData)
	logData.Content = params.Content
	logData.Time = time.Now().Format(logDateFormat)
	var module string
	if s.svrInfoType == svrInfoComm {
		module = s.commSvrInfo.CommName
	} else {
		module = fmt.Sprintf("%s.%s", s.frameSvrInfo.App, s.frameSvrInfo.Server)
	}
	// 需要组装为 {"k0":"v0", "k1":"v1"} 这样的数据类型
	logData.Dimensions = getLogKeyValue(params.Dimensions)
	// log prop 是 固定的四个字段 app.server logName loglevel context_id
	propsStrSlice := []string{
		module, params.Name, hawkLogLevelMap[params.Level], params.ContextID,
	}
	logData.Prop = getLogKeyValue(propsStrSlice)

	return logData
}

// filterParams 过滤不想上报的字段
func filterParams(ignoreList []int, params *HawkLogParams) {
	for _, index := range ignoreList {
		params.Dimensions[index] = "ignore"
	}
}

// getLogKeyValue 将slice里面的数据 组装为 [{"k0":"v0"}, {"k1":"v1"}] 这样的数据类型
func getLogKeyValue(items []string) map[string]string {
	keyValues := make(map[string]string, len(items))
	for index, item := range items {
		keyValues[fmt.Sprintf("k%d", index)] = item
	}
	return keyValues
}

// sendHawkLogData 发送日志
func (s *Instance) sendHawkLogData(attaID, attaToken string, params *HawkLogParams) {
	// 组装数据
	hawkLogData := s.assembleLogData(params)
	byteData, jsonErr := json.Marshal(hawkLogData)
	if jsonErr != nil {
		return
	}

	err := s.withLogChan(params.Level, &logChanData{
		attaToken: attaToken,
		attaID:    attaID,
		logName:   params.Name,
		data:      byteData,
	})

	if err != nil {
		log.Printf("trpc_report_api_go:atta send fail:%v", err)
	}
}

// getConfigWithLog 根据logName获取中控的配置，获取到的是更新后的中控全量信息
func (s *Instance) getConfigWithLog(logName string, version, dimensionsNum int) {
	defer func() {
		// 捕获panic
		if err := recover(); err != nil {
			log.Printf("getConfigWithLog: panic recovery, err:%v", err)
			s.newHawkLogMap.Delete(logName)
		}
	}()
	sdkConfList := make([]*config.HawkLogSdk, 0, 1)
	sdkConf := &config.HawkLogSdk{
		Name:          logName,
		Version:       version,
		DimensionsNum: dimensionsNum,
	}
	sdkConfList = append(sdkConfList, sdkConf)
	err := s.configLooper.updateConfig(&config.ReqBody{
		ConfigInfoLogBody: &config.HawkLogBody{
			Params: sdkConfList,
		},
	})
	if err != nil {
		s.newHawkLogMap.Delete(logName)
	}
}

// logSend 是否上报日志
func (s *Instance) logSend(params *HawkLogParams, rate int) bool {
	// 错误日志都需要上报
	if params.Level >= ErrorLevel {
		return true
	}
	// 含有contextID的日志需要进一步判断是否采样
	if params.ContextID != "" {
		return s.logSampleWithContextID(params.Name, params.ContextID, rate)
	}
	return isLogSend(rate)
}

// logSampleWithContextID 含有contextID的日志进行抽样
func (s *Instance) logSampleWithContextID(name, contextId string, rate int) bool {
	key := fmt.Sprintf("%s_%s", name, contextId)
	// 当前的ContextID 已经被采样，如果没到过期时间,该日志下的所有日志在这段时间都不过滤
	expireTime, ok := s.hawkLogSampleMap.Load(key)
	if !ok || expireTime == nil {
		// 采样
		if !isLogSend(rate) {
			return false
		}
		// 当前contextID 被采样了，则接下来的5分钟内 该日志都不过滤
		s.hawkLogSampleMap.Store(key, time.Now().Unix()+contextIDSampleExpireTime)
		return true
	}
	// 已经过期了，需要删除当前的ContextID 进行重新采样
	if time.Now().Unix() > expireTime.(int64) {
		s.hawkLogSampleMap.Delete(key)
		return false
	}
	return true
}

// AddLogNames 添加日志名称
func (s *Instance) AddLogNames(logNames []string) {
	s.hawkLogNames = append(s.hawkLogNames, logNames...)
	s.configLooper.initNewLog()
}

// isLogSend 日志采样判断
func isLogSend(rate int) bool {
	if rate <= 0 || rate > 100 {
		return false
	}
	// 采样率判断
	return rate >= rand.Intn(100)
}

// withLogChan 将数据写到管道，异步定时上报
func (s *Instance) withLogChan(level HawkLogLevel, logChanData *logChanData) error {
	ret := retSuccess
	select {
	case s.logChannel <- logChanData:
	default:
		ret = retFailure
		select {
		case s.logReportLooper.active <- 1: // 管道满，数据丢弃，兜底激活goroutinue上报数据
		default:
		}
	}

	s.sendDependOnContentNum()
	// 上报007日志自监控
	s.selfMonitorLooper.addSendHawkLogSelfMonitor(ret, len(logChanData.data), logChanData.logName, hawkLogLevelMap[level])

	if ret != retSuccess {
		return ErrorReportLossData
	}
	return nil
}

// logSendDependOnContentNum 管道数量达到汇聚条件时促发汇聚
func (s *Instance) logSendDependOnContentNum() {
	if len(s.logChannel) >= logChannelMaxSize {
		select {
		case s.logReportLooper.active <- 1:
		default:
		}
	}
}
