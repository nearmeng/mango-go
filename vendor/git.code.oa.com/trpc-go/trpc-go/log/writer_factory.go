package log

import (
	"errors"
	"path/filepath"

	"git.code.oa.com/trpc-go/trpc-go/plugin"
)

var (
	// DefaultConsoleWriterFactory 默认的console输出流实现
	DefaultConsoleWriterFactory = &ConsoleWriterFactory{}
	// DefaultFileWriterFactory 默认的file输出流实现
	DefaultFileWriterFactory = &FileWriterFactory{}

	writers = make(map[string]plugin.Factory)
)

// RegisterWriter 注册日志输出writer，支持同时多个日志实现
func RegisterWriter(name string, writer plugin.Factory) {
	writers[name] = writer
}

// GetWriter 获取日志输出writer，不存在返回nil
func GetWriter(name string) plugin.Factory {
	return writers[name]
}

// ConsoleWriterFactory  new console writer instance
type ConsoleWriterFactory struct {
}

// Type 日志插件类型
func (f *ConsoleWriterFactory) Type() string {
	return pluginType
}

// Setup 启动加载配置 并注册console output writer
func (f *ConsoleWriterFactory) Setup(name string, dec plugin.Decoder) error {
	if dec == nil {
		return errors.New("console writer decoder empty")
	}
	decoder, ok := dec.(*Decoder)
	if !ok {
		return errors.New("console writer log decoder type invalid")
	}
	cfg := &OutputConfig{}
	if err := decoder.Decode(&cfg); err != nil {
		return err
	}
	decoder.Core, decoder.ZapLevel = newConsoleCore(cfg)
	return nil
}

// FileWriterFactory  new file writer instance
type FileWriterFactory struct {
}

// Type 日志插件类型
func (f *FileWriterFactory) Type() string {
	return pluginType
}

// Setup 启动加载配置 并注册file output writer
func (f *FileWriterFactory) Setup(name string, dec plugin.Decoder) error {
	if dec == nil {
		return errors.New("file writer decoder empty")
	}
	decoder, ok := dec.(*Decoder)
	if !ok {
		return errors.New("file writer log decoder type invalid")
	}
	if err := f.setupConfig(decoder); err != nil {
		return err
	}
	return nil
}

func (f *FileWriterFactory) setupConfig(decoder *Decoder) error {
	cfg := &OutputConfig{}
	if err := decoder.Decode(&cfg); err != nil {
		return err
	}
	if cfg.WriteConfig.LogPath != "" {
		cfg.WriteConfig.Filename = filepath.Join(cfg.WriteConfig.LogPath, cfg.WriteConfig.Filename)
	}
	if cfg.WriteConfig.RollType == "" {
		cfg.WriteConfig.RollType = RollBySize
	}
	if cfg.WriteConfig.WriteMode == 0 {
		cfg.WriteConfig.WriteMode = WriteFast // 默认极速写模式，性能更好，日志满丢弃，防止阻塞服务
	}
	core, level, err := newFileCore(cfg)
	if err != nil {
		return err
	}
	decoder.Core, decoder.ZapLevel = core, level
	return nil
}
