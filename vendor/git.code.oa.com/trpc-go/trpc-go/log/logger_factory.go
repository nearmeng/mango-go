package log

import (
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"git.code.oa.com/trpc-go/trpc-go/plugin"
)

func init() {
	RegisterWriter(OutputConsole, DefaultConsoleWriterFactory)
	RegisterWriter(OutputFile, DefaultFileWriterFactory)
	Register(defaultLoggerName, NewZapLog(defaultConfig))
	plugin.Register(defaultLoggerName, DefaultLogFactory)
}

const (
	pluginType        = "log"
	defaultLoggerName = "default"
)

var (
	// DefaultLogger 默认的logger实现，初始值为console输出，当框架启动后，通过配置文件初始化后覆盖该值
	DefaultLogger Logger
	// DefaultLogFactory 默认的日志加载实现，用户可以自己实现并替换进来
	DefaultLogFactory = &Factory{}

	mu      sync.RWMutex
	loggers = make(map[string]Logger)
)

// Register 注册日志，支持同时多个日志实现
func Register(name string, logger Logger) {
	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		panic("log: Register logger is nil")
	}
	if _, dup := loggers[name]; dup && name != defaultLoggerName {
		panic("log: Register called twiced for logger name " + name)
	}
	loggers[name] = logger
	if name == defaultLoggerName {
		DefaultLogger = logger
	}
}

// GetDefaultLogger() 默认的logger，通过配置文件key=default来设置, 默认使用console输出
func GetDefaultLogger() Logger {
	mu.RLock()
	l := DefaultLogger
	mu.RUnlock()
	return l
}

// SetLogger 设置默认logger
func SetLogger(logger Logger) {
	mu.Lock()
	DefaultLogger = logger
	mu.Unlock()
}

// Get 通过日志名返回具体的实现 log.Debug使用DefaultLogger打日志，也可以使用 log.Get("name").Debug
func Get(name string) Logger {
	mu.RLock()
	l := loggers[name]
	mu.RUnlock()
	return l
}

// Sync 对注册的所有logger执行Sync动作
func Sync() {
	for _, logger := range loggers {
		_ = logger.Sync()
	}
}

// Decoder log
type Decoder struct {
	OutputConfig *OutputConfig
	Core         zapcore.Core
	ZapLevel     zap.AtomicLevel
}

// Decode 解析writer配置 复制一份
func (d *Decoder) Decode(cfg interface{}) error {
	output, ok := cfg.(**OutputConfig)
	if !ok {
		return fmt.Errorf("decoder config type:%T invalid, not **OutputConfig", cfg)
	}
	*output = d.OutputConfig
	return nil
}

// Factory 日志插件工厂 由框架启动读取配置文件 调用该工厂生成具体日志
type Factory struct {
}

// Type 日志插件类型
func (f *Factory) Type() string {
	return pluginType
}

// Setup 启动加载log配置 并注册日志
func (f *Factory) Setup(name string, dec plugin.Decoder) error {
	if dec == nil {
		return errors.New("log config decoder empty")
	}
	cfg, callerSkip, err := f.setupConfig(dec)
	if err != nil {
		return err
	}
	logger := NewZapLogWithCallerSkip(cfg, callerSkip)
	if logger == nil {
		return errors.New("new zap logger fail")
	}
	Register(name, logger)
	return nil
}

func (f *Factory) setupConfig(configDec plugin.Decoder) (Config, int, error) {
	cfg := Config{}
	if err := configDec.Decode(&cfg); err != nil {
		return nil, 0, err
	}
	if len(cfg) == 0 {
		return nil, 0, errors.New("log config output empty")
	}

	// 如果没有配置caller skip，则默认为2
	callerSkip := 2
	for i := 0; i < len(cfg); i++ {
		if cfg[i].CallerSkip != 0 {
			callerSkip = cfg[i].CallerSkip
		}
	}
	return cfg, callerSkip, nil
}
