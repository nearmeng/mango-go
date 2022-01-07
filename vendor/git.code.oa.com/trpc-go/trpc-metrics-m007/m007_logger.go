package m007

import (
	"errors"
	"strings"

	pcgmonitor "git.code.oa.com/pcgmonitor/trpc_report_api_go"
	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	tlog "git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logPluginName = "m007_log"
	logType       = "log"

	messageKey = "msg"
	levelKey   = "level"
	timeKey    = "time"
	callerKey  = "caller"
)

func init() {
	tlog.RegisterWriter(logPluginName, &logPlugin{})
}

var (
	logInstance = pcgmonitor.NewInstance()
)

// logLevelMap 日志级别的枚举
var logLevelMap = map[string]pcgmonitor.HawkLogLevel{
	"TRACE": pcgmonitor.TraceLevel,
	"DEBUG": pcgmonitor.DebugLevel,
	"INFO":  pcgmonitor.InfoLevel,
	"WARN":  pcgmonitor.WarnLevel,
	"ERROR": pcgmonitor.ErrorLevel,
	"FATAL": pcgmonitor.FatalLevel,
}

// LogConfig 日志配置项
type LogConfig struct {
	AppName        string `yaml:"app"`            // 业务名
	ServerName     string `yaml:"server"`         // 服务名
	IP             string `yaml:"ip"`             // 本机IP
	ContainerName  string `yaml:"containerName"`  // 容器名称
	ContainerSetId string `yaml:"containerSetId"` // 容器SetId
	Version        string `yaml:"version"`        // 应用版本
	PhysicEnv      string `yaml:"physicEnv"`      // 物理环境
	UserEnv        string `yaml:"userEnv"`        // 用户环境
	FrameCode      string `yaml:"frameCode"`      // 架构版本
	DebugLogOpen   bool   `yaml:"debuglogOpen"`   // debuglog输出
	PolarisAddrs   string `yaml:"polarisAddrs"`   // 北极星地址
	PolarisProto   string `yaml:"polarisProto"`   // 北极星协议

	LogName    string   `yaml:"logName"` // 日志名称
	Field      []string // 日志字段,不建议删除字段,可修改字段名称,如果新增字段请在最后新增
	LevelKey   string   `yaml:"level_key"`   //[可选，默认为空]日志级别对应的日志字段，不需可不配置
	MessageKey string   `yaml:"message_key"` // 日志打印包体的对应日志的field
	CallerKey  string   `yaml:"caller_key"`  // 日志输出调用者在以Json输出时的key的名称
	TimeKey    string   `yaml:"time_key"`    // 日志输出时间在以Json输出时的key的名称
}

// Logger  007日志 logger
type Logger struct {
	MessageKey string
	NameKey    string
	LevelKey   string
	CallerKey  string
	TimeKey    string
	Field      []string
}

// logPlugin log plugin
type logPlugin struct{}

// Type 插件类型
func (l *logPlugin) Type() string {
	return logType
}

// Setup 日志插件初始化
func (l *logPlugin) Setup(name string, configDec plugin.Decoder) error {
	// 配置解析, 配置错误依旧返回err, 外部依赖错误降级处理
	decoder, conf, cfg, err := getDecoderAndConf(configDec)
	if err != nil {
		return err
	}

	if len(cfg.LogName) == 0 {
		return errors.New("logName is required")
	}

	fixLogConfig(cfg)

	if err := logInstance.Setup(&pcgmonitor.FrameSvrSetupInfo{
		FrameSvrInfo: pcgmonitor.FrameSvrInfo{
			App:          cfg.AppName,
			Server:       cfg.ServerName,
			IP:           cfg.IP,
			Container:    cfg.ContainerName,
			ConSetId:     cfg.ContainerSetId,
			Version:      cfg.Version,
			PhysicEnv:    cfg.PhysicEnv,
			UserEnv:      cfg.UserEnv,
			FrameCode:    cfg.FrameCode,
			DebugLogOpen: cfg.DebugLogOpen,
			HawkLogNames: []string{cfg.LogName},
		},
		PolarisInfo: pcgmonitor.PolarisInfo{ // 拉007配置时使用, 不填，使用默认值
			Addrs: cfg.PolarisAddrs,
			Proto: cfg.PolarisProto,
		},
	}); err != nil {
		return err
	}
	logger := &Logger{
		MessageKey: cfg.MessageKey,
		LevelKey:   cfg.LevelKey,
		NameKey:    cfg.LogName,
		CallerKey:  cfg.CallerKey,
		TimeKey:    cfg.TimeKey,
		Field:      cfg.Field,
	}

	encoderCfg := zapcore.EncoderConfig{
		LevelKey:       cfg.LevelKey,
		MessageKey:     cfg.MessageKey,
		NameKey:        cfg.LogName,
		CallerKey:      cfg.CallerKey,
		TimeKey:        cfg.TimeKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     tlog.NewTimeEncoder(conf.FormatConfig.TimeFmt),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderCfg)
	c := zapcore.NewCore(
		encoder,
		zapcore.AddSync(logger),
		zap.NewAtomicLevelAt(tlog.Levels[conf.Level]),
	)

	decoder.Core = c

	return nil
}

// Write 写日志
func (l *Logger) Write(p []byte) (n int, err error) {
	msg := make(map[string]string)
	if err := codec.Unmarshal(codec.SerializationTypeJSON, p, &msg); err != nil {
		return 0, err
	}
	// 字段拼接
	params := new(pcgmonitor.HawkLogParams)
	params.Name = l.NameKey
	level, ok := msg[l.LevelKey]
	if !ok {
		return 0, errors.New("log level parse fail")
	}
	params.Level = logLevelMap[strings.ToUpper(level)]
	delete(msg, l.LevelKey)

	content, ok := msg[l.MessageKey]
	if !ok {
		return 0, errors.New("log content parse fail")
	}
	params.Content = content
	delete(msg, l.MessageKey)

	caller, ok := msg[l.CallerKey]
	if !ok {
		return 0, errors.New("log caller parse fail")
	}

	fieldValues := make([]string, 0, len(l.Field))
	fieldValues = append(fieldValues, caller)
	delete(msg, l.CallerKey)
	for _, field := range l.Field {
		if val, ok := msg[field]; ok {
			fieldValues = append(fieldValues, val)
			continue
		}
		if field == l.LevelKey || field == l.MessageKey || field == l.CallerKey || field == l.TimeKey {
			continue
		}
		fieldValues = append(fieldValues, "NULL")
	}

	params.Dimensions = fieldValues

	if err := logInstance.ReportHawkLog(params); err != nil {
		return 0, err
	}

	return len(p), nil
}

// getDecoderAndConf 解析日志配置
func getDecoderAndConf(configDec plugin.Decoder) (*tlog.Decoder, *tlog.OutputConfig, *LogConfig, error) {
	if configDec == nil {
		return nil, nil, nil, errors.New("007 log writer decoder empty")
	}
	decoder, ok := configDec.(*tlog.Decoder)
	if !ok {
		return nil, nil, nil, errors.New("007 log writer log decoder type invalid")
	}

	conf := &tlog.OutputConfig{}
	if err := decoder.Decode(&conf); err != nil {
		return nil, nil, nil, err
	}

	var cfg LogConfig
	if err := conf.RemoteConfig.Decode(&cfg); err != nil {
		return nil, nil, nil, err
	}
	return decoder, conf, &cfg, nil
}

// fixlogConfig 修复配置项
func fixLogConfig(logConf *LogConfig) {
	fixLogEnvConfig(logConf)

	fixLogServerConfig(logConf)

	fixLogKey(logConf)

	if len(logConf.FrameCode) == 0 {
		logConf.FrameCode = "trpc"
	}
}

func fixLogKey(logConf *LogConfig) {
	if len(logConf.MessageKey) == 0 {
		logConf.MessageKey = messageKey
	}
	if len(logConf.LevelKey) == 0 {
		logConf.LevelKey = levelKey
	}
	if len(logConf.TimeKey) == 0 {
		logConf.TimeKey = timeKey
	}
	if len(logConf.CallerKey) == 0 {
		logConf.CallerKey = callerKey
	}
}

func fixLogEnvConfig(logConf *LogConfig) {
	if len(logConf.ContainerName) == 0 {
		logConf.ContainerName = trpc.GlobalConfig().Global.ContainerName
	}
	if len(logConf.ContainerSetId) == 0 {
		logConf.ContainerSetId = trpc.GlobalConfig().Global.FullSetName
	}
	if len(logConf.PhysicEnv) == 0 {
		logConf.PhysicEnv = trpc.GlobalConfig().Global.Namespace
	}
	if len(logConf.UserEnv) == 0 {
		logConf.UserEnv = trpc.GlobalConfig().Global.EnvName
	}
}

func fixLogServerConfig(logConf *LogConfig) {
	if len(logConf.AppName) == 0 {
		logConf.AppName = trpc.GlobalConfig().Server.App
	}
	if len(logConf.ServerName) == 0 {
		logConf.ServerName = trpc.GlobalConfig().Server.Server
	}
	if len(logConf.IP) == 0 {
		logConf.IP = trpc.GlobalConfig().Global.LocalIP
	}
}
