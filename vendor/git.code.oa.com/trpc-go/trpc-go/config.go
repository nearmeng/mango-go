package trpc

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/overloadctrl"
	"git.code.oa.com/trpc-go/trpc-go/plugin"

	yaml "gopkg.in/yaml.v3"
)

// ServerConfigPath trpc服务配置文件路径，默认是在启动进程当前目录下的trpc_go.yaml，可通过命令行参数 -conf 指定
var ServerConfigPath = defaultConfigPath

const (
	defaultConfigPath  = "./trpc_go.yaml"
	defaultIdleTimeout = 60000 // 单位 ms
)

// serverConfigPath 获取服务启动配置文件路径
//	最高优先级：服务主动修改ServerConfigPath变量
//	第二优先级：服务通过--conf或者-conf传入配置文件路径
//	第三优先级：默认路径./trpc_go.yaml
func serverConfigPath() string {
	if ServerConfigPath == defaultConfigPath {
		flag.StringVar(&ServerConfigPath, "conf", defaultConfigPath, "server config path")
		flag.Parse()
	}
	return ServerConfigPath
}

// Config trpc配置实现，分四大块：全局配置global，服务端配置server，客户端配置client，插件配置plugins
type Config struct {
	Global struct {
		Namespace      string `yaml:"namespace"`
		EnvName        string `yaml:"env_name"`
		ContainerName  string `yaml:"container_name"`
		LocalIP        string `yaml:"local_ip"`
		EnableSet      string `yaml:"enable_set"`                 // Y/N，是否启用Set分组，默认N
		FullSetName    string `yaml:"full_set_name"`              // set分组的名字，三段式：[set名].[set地区].[set组名]
		ReadBufferSize *int   `yaml:"read_buffer_size,omitempty"` // 网络收包缓冲区大小(单位B)：<=0表示禁用，不配使用默认值
	}
	Server struct {
		App      string
		Server   string
		BinPath  string `yaml:"bin_path"`
		DataPath string `yaml:"data_path"`
		ConfPath string `yaml:"conf_path"`
		Admin    struct {
			IP           string `yaml:"ip"` // 要绑定的网卡地址, 如127.0.0.1
			Nic          string
			Port         uint16 `yaml:"port"`          // 要绑定的端口号，如80，默认值9028
			ReadTimeout  int    `yaml:"read_timeout"`  // ms. 请求被接受到请求信息被完全读取的超时时间设置，防止慢客户端
			WriteTimeout int    `yaml:"write_timeout"` // ms. 处理的超时时间
			EnableTLS    bool   `yaml:"enable_tls"`    // 是否启用tls
		}
		Network       string           // 针对所有service的network 默认tcp
		Protocol      string           // 针对所有service的protocol 默认trpc
		Filter        []string         // 针对所有service的拦截器
		Service       []*ServiceConfig // 单个service服务的配置
		CloseWaitTime int              `yaml:"close_wait_time"` // ms. 关闭服务时,反注册后到真正停止服务之间的等待时间,来支持无损更新
	}
	Client  ClientConfig
	Plugins plugin.Config
}

// ServiceConfig 每个service的配置项，一个服务进程可以支持多个service
type ServiceConfig struct {
	DisableRequestTimeout bool     `yaml:"disable_request_timeout"` // 禁用继承上游的超时时间
	IP                    string   `yaml:"ip"`                      // 监听地址 ip
	Name                  string   // 配置文件定义的用于名字服务的service name:trpc.app.server.service
	Nic                   string   // 监听网卡, 默认情况下由运维分配ip port，此处留空即可，没有分配的情况下 可以支持配置监听网卡
	Port                  uint16   // 监听端口 port
	Address               string   // 监听地址 兼容非ipport模式，有配置address则忽略ipport，没有配置则使用ipport
	Network               string   // 监听网络类型 tcp udp
	Protocol              string   // 业务协议trpc
	Timeout               int      // handler最长处理时间 1s
	Idletime              int      // server端连接空闲超时时间，单位ms，默认 1m
	Registry              string   // 使用哪个注册中心 polaris
	Filter                []string // service拦截器
	TLSKey                string   `yaml:"tls_key"`                // server秘钥
	TLSCert               string   `yaml:"tls_cert"`               // server证书
	CACert                string   `yaml:"ca_cert"`                // ca证书，用于校验client证书，以更严格识别客户端的身份，限制客户端的访问
	ServerAsync           *bool    `yaml:"server_async,omitempty"` // 启用服务器异步处理
	MaxRoutines           int      `yaml:"max_routines"`           // 服务器异步处理模式下，最大协程数限制
	Writev                *bool    `yaml:"writev,omitempty"`       // 启用服务器批量发包(writev系统调用)

	OverloadCtrl  overloadctrl.Impl `yaml:"overload_ctrl"`  // 过载保护
	OverloadCtrls []string          `yaml:"overload_ctrls"` // 与旧版本保持兼容，只使用其第一项配置，如果有的话
}

// ClientConfig 后端服务配置
type ClientConfig struct {
	Network        string   // 针对所有后端的network 默认tcp
	Protocol       string   // 针对所有后端的protocol 默认trpc
	Filter         []string // 针对所有后端的拦截器
	Namespace      string   // 针对所有后端的namespace
	Timeout        int
	Discovery      string
	Loadbalance    string
	Circuitbreaker string
	Service        []*client.BackendConfig // 单个后端请求的配置
}

// trpc server配置信息，由框架启动后解析yaml文件并赋值
var globalConfig atomic.Value

func init() {
	globalConfig.Store(defaultConfig())
}

func defaultConfig() *Config {
	cfg := &Config{}
	cfg.Global.EnableSet = "N"
	cfg.Server.Network = "tcp"
	cfg.Server.Protocol = "trpc"
	cfg.Client.Network = "tcp"
	cfg.Client.Protocol = "trpc"
	return cfg
}

// GlobalConfig 获取全局配置对象
func GlobalConfig() *Config {
	return globalConfig.Load().(*Config)
}

// SetGlobalConfig 设置全局配置对象
func SetGlobalConfig(cfg *Config) {
	globalConfig.Store(cfg)
}

// LoadGlobalConfig 从配置文件加载配置，并设置到全局结构里面
func LoadGlobalConfig(configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	SetGlobalConfig(cfg)
	return nil
}

// LoadConfig 从配置文件加载配置, 并填充好默认值
func LoadConfig(configPath string) (*Config, error) {
	cfg, err := parseConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}
	if err := RepairConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseConfigFromFile(configPath string) (*Config, error) {
	buf, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	// 支持环境变量
	buf = []byte(ExpandEnv(string(buf)))

	cfg := defaultConfig()
	if err := yaml.Unmarshal(buf, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Setup 根据配置进行初始化工作, 包括注册client配置和执行插件初始化工作
func Setup(cfg *Config) error {
	// 将后端配置注册到框架中，由框架在调用时自动加载配置
	for _, backendCfg := range cfg.Client.Service {
		client.RegisterClientConfig(backendCfg.Callee, backendCfg)
	}

	// * 针对所有后端的通用配置参数
	client.RegisterClientConfig("*", &client.BackendConfig{
		Network:        cfg.Client.Network,
		Protocol:       cfg.Client.Protocol,
		Namespace:      cfg.Client.Namespace,
		Timeout:        cfg.Client.Timeout,
		Filter:         cfg.Client.Filter,
		Discovery:      cfg.Client.Discovery,
		Loadbalance:    cfg.Client.Loadbalance,
		Circuitbreaker: cfg.Client.Circuitbreaker,
	})

	// 装载插件
	if cfg.Plugins != nil {
		if err := cfg.Plugins.Setup(); err != nil {
			return err
		}
		plugin.SetupFinished()
	}

	return nil
}

// RepairConfig 修复配置数据，填充默认值
func RepairConfig(cfg *Config) error {
	// nic -> ip
	if err := repairServiceIPWithNic(cfg); err != nil {
		return err
	}
	// 设置默认读缓冲区大小
	if cfg.Global.ReadBufferSize == nil {
		readerSize := codec.DefaultReaderSize
		cfg.Global.ReadBufferSize = &readerSize
	}
	codec.SetReaderSize(*cfg.Global.ReadBufferSize)

	// protocol network ip empty
	for _, serviceCfg := range cfg.Server.Service {
		setDefault(&serviceCfg.Protocol, cfg.Server.Protocol)
		setDefault(&serviceCfg.Network, cfg.Server.Network)
		setDefault(&serviceCfg.IP, cfg.Global.LocalIP)
		setDefault(&serviceCfg.Address, net.JoinHostPort(serviceCfg.IP, strconv.Itoa(int(serviceCfg.Port))))

		// 默认采用异步处理
		if serviceCfg.ServerAsync == nil {
			enableServerAsync := true
			serviceCfg.ServerAsync = &enableServerAsync
		}
		// 默认不开启批量发包
		if serviceCfg.Writev == nil {
			enableWritev := false
			serviceCfg.Writev = &enableWritev
		}
		if serviceCfg.Idletime == 0 {
			serviceCfg.Idletime = defaultIdleTimeout
		}
	}

	setDefault(&cfg.Client.Namespace, cfg.Global.Namespace)
	for _, backendCfg := range cfg.Client.Service {
		repairClientConfig(backendCfg, &cfg.Client)
	}
	return nil
}

// repairServiceIPWithNic 如果没有设置Service监听的IP，则以监听网卡对应的IP来配置
func repairServiceIPWithNic(cfg *Config) error {
	for index, item := range cfg.Server.Service {
		if item.IP == "" {
			ip := GetIP(item.Nic)
			if ip == "" && item.Nic != "" {
				return fmt.Errorf("can't find service IP by the NIC: %s", item.Nic)
			}
			cfg.Server.Service[index].IP = ip
		}
		setDefault(&cfg.Global.LocalIP, item.IP)
	}

	if cfg.Server.Admin.IP == "" {
		ip := GetIP(cfg.Server.Admin.Nic)
		if ip == "" && cfg.Server.Admin.Nic != "" {
			return fmt.Errorf("can't find admin IP by the NIC: %s", cfg.Server.Admin.Nic)
		}
		cfg.Server.Admin.IP = ip
	}
	return nil
}

func repairClientConfig(backendCfg *client.BackendConfig, clientCfg *ClientConfig) {
	// 默认以proto的service name为key来映射客户端配置，一般proto的service name和后端的service name是相同的，所以默认可不配置
	setDefault(&backendCfg.Callee, backendCfg.ServiceName)
	setDefault(&backendCfg.ServiceName, backendCfg.Callee)
	setDefault(&backendCfg.Namespace, clientCfg.Namespace)
	setDefault(&backendCfg.Network, clientCfg.Network)
	setDefault(&backendCfg.Protocol, clientCfg.Protocol)
	if backendCfg.Target == "" {
		setDefault(&backendCfg.Discovery, clientCfg.Discovery)
		setDefault(&backendCfg.Loadbalance, clientCfg.Loadbalance)
		setDefault(&backendCfg.Circuitbreaker, clientCfg.Circuitbreaker)
	}
	if backendCfg.Timeout <= 0 {
		backendCfg.Timeout = clientCfg.Timeout
	}
	backendCfg.Filter = Deduplicate(clientCfg.Filter, backendCfg.Filter) // 全局filter在前，且去重
}

// getMillisecond trpc多语言框架统一以ms为单位配置所有时间字段
func getMillisecond(sec int) time.Duration {
	return time.Millisecond * time.Duration(sec)
}

// setDefault 目标字段为空时则填充默认值
func setDefault(dst *string, def string) {
	if dst != nil && *dst == "" {
		*dst = def
	}
}

// UnmarshalYAML 构建过载保护实例。为了防止隐式地丢失用户配置，我们必须对 overload_ctrls 字段做向后兼容。
func (cfg *ServiceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type tmp ServiceConfig
	if err := unmarshal((*tmp)(cfg)); err != nil {
		return err
	}

	// 保证兼容性
	if len(cfg.OverloadCtrls) > 1 {
		return errors.New("multiple overload controllers are not supported any more")
	}
	if len(cfg.OverloadCtrls) == 1 && cfg.OverloadCtrl.Builder != "" {
		return errors.New("both overload_ctrl and overload_ctrls are set")
	}
	if len(cfg.OverloadCtrls) == 1 {
		cfg.OverloadCtrl.Builder = cfg.OverloadCtrls[0]
	}

	return cfg.OverloadCtrl.Build(overloadctrl.GetServer, &overloadctrl.ServiceMethodInfo{
		ServiceName: cfg.Name,
		MethodName:  overloadctrl.AnyMethod,
	})
}
