package registry

import (
	"errors"
	"strings"
	"time"

	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/config"
	plog "git.code.oa.com/polaris/polaris-go/pkg/log"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
)

const (
	defaultConnectTimeout = time.Second
	defaultMessageTimeout = time.Second
	defaultProtocol       = "grpc"
)

// FactoryConfig 配置
type FactoryConfig struct {
	EnableRegister     bool           `yaml:"register_self"`
	Protocol           string         `yaml:"protocol"`
	HeartbeatInterval  int            `yaml:"heartbeat_interval"`
	Services           []Service      `yaml:"service"`
	Debug              bool           `yaml:"debug"`
	AddressList        string         `yaml:"address_list"`
	JoinPoint          *string        `yaml:"join_point"`
	ClusterService     ClusterService `yaml:"cluster_service"`
	ConnectTimeout     int            `yaml:"connect_timeout"`
	MessageTimeout     *time.Duration `yaml:"message_timeout"`
	DisableHealthCheck bool           `yaml:"disable_health_check"`
}

// ClusterService 集群服务
type ClusterService struct {
	Discover    string `yaml:"discover"`
	HealthCheck string `yaml:"health_check"`
	Monitor     string `yaml:"monitor"`
}

// Service 服务配置
type Service struct {
	Namespace   string            `yaml:"namespace"`
	ServiceName string            `yaml:"name"`
	Token       string            `yaml:"token"`
	InstanceID  string            `yaml:"instance_id"`
	Weight      int               `yaml:"weight"`
	BindAddress string            `yaml:"bind_address"`
	MetaData    map[string]string `yaml:"metadata"`
}

func init() {
	plugin.Register("polaris", &RegistryFactory{})
}

// RegistryFactory 注册工厂
type RegistryFactory struct {
}

// Type 返回注册类型
func (f *RegistryFactory) Type() string {
	return "registry"
}

// Setup 启动加载配置 并注册日志
func (f *RegistryFactory) Setup(name string, configDec plugin.Decoder) error {
	if configDec == nil {
		return errors.New("registry config decoder empty")
	}
	conf := &FactoryConfig{}
	if err := configDec.Decode(conf); err != nil {
		return err
	}
	if conf.Debug {
		log.Debug("set polaris log level debug")
		plog.GetBaseLogger().SetLogLevel(plog.DebugLog)
	}
	return register(conf)
}

func newProvider(cfg *FactoryConfig) (api.ProviderAPI, error) {
	var c *config.ConfigurationImpl
	if len(cfg.AddressList) > 0 {
		addressList := strings.Split(cfg.AddressList, ",")
		c = config.NewDefaultConfiguration(addressList)
	} else {
		c = config.NewDefaultConfigurationWithDomain()
	}
	// 配置 cluster
	if cfg.ClusterService.Discover != "" {
		c.Global.GetSystem().GetDiscoverCluster().SetService(cfg.ClusterService.Discover)
	}
	if cfg.ClusterService.HealthCheck != "" {
		c.Global.GetSystem().GetHealthCheckCluster().SetService(cfg.ClusterService.HealthCheck)
	}
	if cfg.ClusterService.Monitor != "" {
		c.Global.GetSystem().GetMonitorCluster().SetService(cfg.ClusterService.Monitor)
	}
	// 设置 joinPoint，会覆盖address_list和 cluster 配置
	if cfg.JoinPoint != nil {
		c.GetGlobal().GetServerConnector().SetJoinPoint(*cfg.JoinPoint)
	}
	if cfg.Protocol == "" {
		cfg.Protocol = defaultProtocol
	}
	c.Global.ServerConnector.Protocol = cfg.Protocol
	if cfg.ConnectTimeout != 0 {
		c.GetGlobal().GetServerConnector().SetConnectTimeout(time.Duration(cfg.ConnectTimeout) * time.Millisecond)
	} else {
		c.GetGlobal().GetServerConnector().SetConnectTimeout(defaultConnectTimeout)
	}
	// 设置消息超时时间
	messageTimeout := defaultMessageTimeout
	if cfg.MessageTimeout != nil {
		messageTimeout = *cfg.MessageTimeout
	}
	c.GetGlobal().GetServerConnector().SetMessageTimeout(messageTimeout)
	provider, err := api.NewProviderAPIByConfig(c)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func register(conf *FactoryConfig) error {
	provider, err := newProvider(conf)
	if err != nil {
		return err
	}
	for _, service := range conf.Services {
		cfg := &Config{
			Protocol:           conf.Protocol,
			EnableRegister:     conf.EnableRegister,
			HeartBeat:          conf.HeartbeatInterval / 1000,
			ServiceName:        service.ServiceName,
			Namespace:          service.Namespace,
			ServiceToken:       service.Token,
			InstanceID:         service.InstanceID,
			Metadata:           service.MetaData,
			BindAddress:        service.BindAddress,
			DisableHealthCheck: conf.DisableHealthCheck,
		}
		reg, err := newRegistry(provider, cfg)
		if err != nil {
			return err
		}
		registry.Register(service.ServiceName, reg)
	}
	return nil
}
