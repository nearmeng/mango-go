package client

import (
	"sync/atomic"

	"git.code.oa.com/trpc-go/trpc-go/config"
	"git.code.oa.com/trpc-go/trpc-go/overloadctrl"
)

// BackendConfig 后端配置参数, 框架提供替换后端配置参数能力，可以由第三方注册进来，默认为空
type BackendConfig struct {
	// Callee 对端服务协议文件的callee service name
	// 配置文件以这个为key来设置参数, 一般callee和下面这个servicename是一致的
	// 可以为空，也可以支持用户随便自定义下游service name
	Callee               string `yaml:"callee"`
	ServiceName          string `yaml:"name"`                  // 对端服务真实名字服务的service name
	EnvName              string `yaml:"env_name"`              // 设置下游服务多环境的环境名
	SetName              string `yaml:"set_name"`              // 设置下游服务set名
	DisableServiceRouter bool   `yaml:"disable_servicerouter"` // 单个client禁用服务路由
	Namespace            string // 对端服务环境 正式环境 测试环境

	OverloadCtrl overloadctrl.Impl `yaml:"overload_ctrl"` // 过载保护

	Target   string // 默认使用北极星，一般不用配置，cl5://sid
	Password string

	Discovery      string
	Loadbalance    string
	Circuitbreaker string

	Network  string // tcp udp
	Timeout  int    // 单位 ms
	Protocol string // trpc

	Serialization *int // 序列化方式,因为默认值0已经用于pb了，所以通过指针来判断是否配置
	Compression   int  // 压缩方式

	TLSKey        string `yaml:"tls_key"`         // client秘钥
	TLSCert       string `yaml:"tls_cert"`        // client证书
	CACert        string `yaml:"ca_cert"`         // ca证书，用于校验server证书，调用tls服务，如https server
	TLSServerName string `yaml:"tls_server_name"` // client校验server服务名，调用https时，默认为hostname

	Filter []string
}

// UnmarshalYAML sets default values for BackendConfig on yaml unmarshal.
func (cfg *BackendConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// 额外定义一个未实现 UnmarshalYAML 的新 type，直接 unmarshal 会造成无限递归，最终栈溢出。
	type tmp BackendConfig
	if err := unmarshal((*tmp)(cfg)); err != nil {
		return err
	}

	// 这段逻辑见 repairClientConfig 中对 Callee 和 ServiceName 的修复。
	name := cfg.ServiceName
	if name == "" {
		name = cfg.Callee
	}

	return cfg.OverloadCtrl.Build(overloadctrl.GetClient, &overloadctrl.ServiceMethodInfo{
		ServiceName: name,
		MethodName:  overloadctrl.AnyMethod,
	})
}

var (
	defaultBackendConf = &BackendConfig{
		Network:  "tcp",
		Protocol: "trpc",
	}
	defaultClientConfig = make(map[string]*BackendConfig) // client proto service name => client backend config
	clientConfig        = atomic.Value{}
)

func init() {
	RegisterConfig(defaultClientConfig)
}

// DefaultClientConfig 获取后端调用配置信息，由业务配置解析并赋值更新，框架读取该结构
func DefaultClientConfig() map[string]*BackendConfig {
	return clientConfig.Load().(map[string]*BackendConfig)
}

// LoadClientConfig 通过本地配置文件路径解析业务配置并注册到框架中
func LoadClientConfig(path string, opts ...config.LoadOption) error {
	conf, err := config.DefaultConfigLoader.Load(path, opts...)
	if err != nil {
		return err
	}
	tmp := make(map[string]*BackendConfig)
	if err := conf.Unmarshal(tmp); err != nil {
		return err
	}
	RegisterConfig(tmp)
	return nil
}

// Config 通过对端协议文件的callee service name获取后端配置
func Config(serviceName string) *BackendConfig {
	def := DefaultClientConfig()
	if len(def) == 0 {
		return defaultBackendConf
	}
	conf, ok := def[serviceName]
	if !ok {
		conf, ok = def["*"]
		if !ok {
			return defaultBackendConf
		}
	}
	return conf
}

// RegisterConfig 业务自己解析完配置后 全局替换注册后端配置信息
func RegisterConfig(conf map[string]*BackendConfig) {
	clientConfig.Store(conf)
}

// RegisterClientConfig 业务自己解析完配置后 注册单个后端配置信息 不可并发调用
func RegisterClientConfig(calleeServiceName string, conf *BackendConfig) {
	DefaultClientConfig()[calleeServiceName] = conf
}
