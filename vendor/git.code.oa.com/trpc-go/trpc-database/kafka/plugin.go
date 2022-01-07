package kafka

import (
	"fmt"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
	"github.com/Shopify/sarama"
)

const (
	pluginType = "database"
	pluginName = "kafka"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Config kafka代理配置结构体声明
type Config struct {
	MaxRequestSize  int32 `yaml:"max_request_size"`  // 全局最大请求体大小
	MaxResponseSize int32 `yaml:"max_response_size"` // 全局最大响应体大小
	RewriteLog      bool  `yaml:"rewrite_log"`       // 是否将日志重写到log中
}

// Plugin proxy 插件默认初始化, 用于加载kafka代理连接参数配置
type Plugin struct{}

// Type 插件类型
func (k *Plugin) Type() string {
	return pluginType
}

// Setup 插件初始化
func (k *Plugin) Setup(name string, configDesc plugin.Decoder) (err error) {
	var config Config
	if err = configDesc.Decode(&config); err != nil {
		return
	}
	if config.MaxRequestSize > 0 {
		sarama.MaxRequestSize = config.MaxRequestSize
	}
	if config.MaxResponseSize > 0 {
		sarama.MaxResponseSize = config.MaxResponseSize
	}
	if config.RewriteLog {
		sarama.Logger = LogReWriter{}
	}
	fmt.Println(config)

	return nil
}

// LogReWriter 重定向日志
type LogReWriter struct {
}

// Print sarama.Logger接口
func (LogReWriter) Print(v ...interface{}) {
	log.Info(v...)
}

// Printf sarama.Logger接口
func (LogReWriter) Printf(format string, v ...interface{}) {
	log.Infof(format, v...)
}

// Println sarama.Logger接口
func (LogReWriter) Println(v ...interface{}) {
	log.Info(v...)
}
