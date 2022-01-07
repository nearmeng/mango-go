package m007

import (
	"context"

	pcgmonitor "git.code.oa.com/pcgmonitor/trpc_report_api_go"
	trpc "git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/metrics"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
)

const (
	pluginType = "metrics"
	pluginName = "m007"
	m007Etx3   = "m007_etx3"
)

func init() {
	plugin.Register(pluginName, &m007Plugin{})
}

// Config 007监控配置项
type Config struct {
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

	PolarisAddrs string `yaml:"polarisAddrs"` // 北极星地址
	PolarisProto string `yaml:"polarisProto"` // 北极星协议
}

// m007Plugin 007监控插件工厂，实例化007插件，注册metrics，模调拦截器
type m007Plugin struct{}

// Type m007Plugin插件类型
func (p *m007Plugin) Type() string {
	return pluginType
}

// Setup m007Plugin插件初始化
func (p *m007Plugin) Setup(name string, decoder plugin.Decoder) error {

	cfg := Config{}

	err := decoder.Decode(&cfg)
	if err != nil {
		log.Errorf("trpc-metrics-m007:conf Decode error:%v", err)
		return err
	}
	fixM007Config(&cfg)

	err = pcgmonitor.Setup(&pcgmonitor.FrameSvrSetupInfo{
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
		},
		PolarisInfo: pcgmonitor.PolarisInfo{ // 拉007配置时使用, 不填，使用默认值
			Addrs: cfg.PolarisAddrs,
			Proto: cfg.PolarisProto,
		},
	})
	// Setup失败注册一个空filter, 直接返回nil不阻塞主流程
	if err != nil {
		log.Errorf("trpc-metrics-m007:pcgmonitor.Setup error:%v", err)
		filter.Register(name, filter.NoopFilter, filter.NoopFilter)
		return nil
	}

	// 注册metrics
	metrics.RegisterMetricsSink(&M007Sink{})

	// 注册主调、被调
	filter.Register(name, PassiveModuleCallServerFilter, ActiveModuleCallClientFilter)

	return nil
}

// fixM007Config 修复配置项
func fixM007Config(m007Conf *Config) {
	fixServerConfig(m007Conf)

	fixEnvConfig(m007Conf)

	if len(m007Conf.FrameCode) == 0 {
		m007Conf.FrameCode = "trpc"
	}
}

func fixEnvConfig(m007Conf *Config) {
	if len(m007Conf.ContainerName) == 0 {
		m007Conf.ContainerName = trpc.GlobalConfig().Global.ContainerName
	}
	if len(m007Conf.ContainerSetId) == 0 {
		m007Conf.ContainerSetId = trpc.GlobalConfig().Global.FullSetName
	}
	if len(m007Conf.PhysicEnv) == 0 {
		m007Conf.PhysicEnv = trpc.GlobalConfig().Global.Namespace
	}
	if len(m007Conf.UserEnv) == 0 {
		m007Conf.UserEnv = trpc.GlobalConfig().Global.EnvName
	}
}

func fixServerConfig(m007Conf *Config) {
	if len(m007Conf.AppName) == 0 {
		m007Conf.AppName = trpc.GlobalConfig().Server.App
	}
	if len(m007Conf.ServerName) == 0 {
		m007Conf.ServerName = trpc.GlobalConfig().Server.Server
	}
	if len(m007Conf.IP) == 0 {
		m007Conf.IP = trpc.GlobalConfig().Global.LocalIP
	}
}

// SetExtensionDimension 设置007主被调监控的拓展字段
func SetExtensionDimension(ctx context.Context, value string) {
	msg := trpc.Message(ctx)
	meta := msg.CommonMeta()
	if meta == nil {
		meta = codec.CommonMeta{}
	}
	meta[m007Etx3] = value
	msg.WithCommonMeta(meta)
}

// getExtensionDimension 获取007主被调监控的拓展字段
func getExtensionDimension(msg codec.Msg) string {
	meta := msg.CommonMeta()
	if meta == nil {
		return ""
	}
	val, ok := meta[m007Etx3].(string)
	if !ok {
		return ""
	}
	return val
}
