package debuglog

import (
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
)

const (
	pluginName = "debuglog"
	pluginType = "tracing"
)

func init() {
	plugin.Register(pluginName, &Plugin{})
}

// Plugin debuglog  trpc 插件实现
type Plugin struct {
}

// Type debuglog trpc插件类型
func (p *Plugin) Type() string {
	return pluginType
}

// Config debuglog插件配置
type Config struct {
	LogType       string `yaml:"log_type"`
	ServerLogType string `yaml:"server_log_type"`
	ClientLogType string `yaml:"client_log_type"`
	Exclude       []*ExcludeItem
}

// ExcludeItem 排除选项
type ExcludeItem struct {
	Method  string
	Retcode int
}

func getLogFunc(t string) LogFunc {
	switch t {
	case "simple":
		return SimpleLogFunc
	case "prettyjson":
		return PrettyJSONLogFunc
	case "json":
		return JSONLogFunc
	default:
		return DefaultLogFunc
	}
}

// Setup debuglog实例初始化
func (p *Plugin) Setup(name string, configDec plugin.Decoder) error {

	var conf Config
	err := configDec.Decode(&conf)
	if err != nil {
		return err
	}

	var serverOpt []Option
	var clientOpt []Option

	serverLogType := conf.LogType
	if conf.ServerLogType != "" {
		serverLogType = conf.ServerLogType
	}
	serverOpt = append(serverOpt, WithLogFunc(getLogFunc(serverLogType)))

	clientLogType := conf.LogType
	if conf.ClientLogType != "" {
		clientLogType = conf.ClientLogType
	}
	clientOpt = append(clientOpt, WithLogFunc(getLogFunc(clientLogType)))

	for _, ex := range conf.Exclude {
		serverOpt = append(serverOpt, WithExclude(ex))
		clientOpt = append(clientOpt, WithExclude(ex))
	}

	filter.Register(pluginName, ServerFilter(serverOpt...), ClientFilter(clientOpt...))

	return nil
}
