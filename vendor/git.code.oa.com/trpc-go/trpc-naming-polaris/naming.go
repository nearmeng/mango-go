package naming

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/config"
	plog "git.code.oa.com/polaris/polaris-go/pkg/log"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	"git.code.oa.com/polaris/polaris-go/plugin/circuitbreaker/errorcount"
	"git.code.oa.com/polaris/polaris-go/plugin/circuitbreaker/errorrate"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/circuitbreaker"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/discovery"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/loadbalance"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/selector"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/servicerouter"

	// 初始化注册模块
	_ "git.code.oa.com/trpc-go/trpc-naming-polaris/registry"
)

func init() {
	plugin.Register("polaris", &SelectorFactory{})
}

// Config 框架配置
type Config struct {
	Debug               bool                   `yaml:"debug"`
	Default             *bool                  `yaml:"default"`
	Protocol            string                 `yaml:"protocol"`
	ReportTimeout       *time.Duration         `yaml:"report_timeout"`
	EnableServiceRouter *bool                  `yaml:"enable_servicerouter"`
	EnableCanary        *bool                  `yaml:"enable_canary"`
	PersistDir          *string                `yaml:"persistDir"`
	ServiceExpireTime   *time.Duration         `yaml:"service_expire_time"`
	LogDir              *string                `yaml:"log_dir"`
	Timeout             int                    `yaml:"timeout"`
	ConnectTimeout      int                    `yaml:"connect_timeout"`
	MessageTimeout      *time.Duration         `yaml:"message_timeout"`
	JoinPoint           *string                `yaml:"join_point"`
	AddressList         string                 `yaml:"address_list"`
	Discovery           DiscoveryConfig        `yaml:"discovery"`
	Loadbalance         LoadbalanceConfig      `yaml:"loadbalance"`
	CircuitBreaker      CircuitBreakerConfig   `yaml:"circuitbreaker"`
	ServiceRouter       ServiceRouterConfig    `yaml:"service_router"`
	ClusterService      ClusterService         `yaml:"cluster_service"`
	OutlierDetection    OutlierDetectionConfig `yaml:"outlierDetection"`
	EnableTransMeta     bool                   `yaml:"enable_trans_meta"`
}

// ServiceRouterConfig
type ServiceRouterConfig struct {
	// NearbyMatchLevel 就近路由的最小匹配级别, 包括region(大区)、zone(区域)、campus(园区), 默认为zone
	NearbyMatchLevel string `yaml:"nearby_matchlevel"`
}

// DiscoveryConfig 配置
type DiscoveryConfig struct {
	RefreshInterval int `yaml:"refresh_interval"`
}

// LoadbalaceConfig
type LoadbalanceConfig struct {
	Name []string `yaml:"name"` // 负载均衡类型
}

// CircuitBreakerConfig
type CircuitBreakerConfig struct {
	CheckPeriod               *time.Duration `yaml:"checkPeriod"`
	RequestCountAfterHalfOpen *int           `yaml:"requestCountAfterHalfOpen"`
	SleepWindow               *time.Duration `yaml:"sleepWindow"`
	SuccessCountAfterHalfOpen *int           `yaml:"successCountAfterHalfOpen"`
	Chain                     []string       `yaml:"chain"`
	ErrorCount                *struct {
		ContinuousErrorThreshold *int           `yaml:"continuousErrorThreshold"`
		MetricNumBuckets         *int           `yaml:"metricNumBuckets"`
		MetricStatTimeWindow     *time.Duration `yaml:"metricStatTimeWindow"`
	} `yaml:"errorCount"`
	ErrorRate *struct {
		ErrorRateThreshold     *float64       `yaml:"errorRateThreshold"`
		MetricNumBuckets       *int           `yaml:"metricNumBuckets"`
		MetricStatTimeWindow   *time.Duration `yaml:"metricStatTimeWindow"`
		RequestVolumeThreshold *int           `yaml:"requestVolumeThreshold"`
	} `yaml:"errorRate"`
}

// OutlierDetectionConfig 熔断探活配置
type OutlierDetectionConfig struct {
	Enable      *bool          `yaml:"enable"`      //是否启动熔断
	CheckPeriod *time.Duration `yaml:"checkPeriod"` //定时探测周期
}

// ClusterService 集群服务
type ClusterService struct {
	Discover    string `yaml:"discover"`
	HealthCheck string `yaml:"health_check"`
	Monitor     string `yaml:"monitor"`
}

// SelectorFactory
type SelectorFactory struct{}

// Type 插件类型
func (f *SelectorFactory) Type() string {
	return "selector"
}

func (conf *Config) getSetDefault() bool {
	setDefault := true
	if conf.Default != nil {
		setDefault = *conf.Default
	}
	return setDefault
}

func (conf *Config) getEnableCanary() bool {
	var isEnable bool
	if conf.EnableCanary != nil {
		isEnable = *conf.EnableCanary
	}
	return isEnable
}

func (conf *Config) getEnableServiceRouter() bool {
	isEnable := true
	if conf.EnableServiceRouter != nil {
		isEnable = *conf.EnableServiceRouter
	}
	return isEnable
}

func (conf *Config) setLog() {
	if conf.LogDir != nil {
		api.ConfigBaseLogger(*conf.LogDir, plog.DefaultBaseLogLevel)
		api.ConfigDetectLogger(*conf.LogDir, plog.DefaultDetectLogLevel)
		api.ConfigStatLogger(*conf.LogDir, plog.DefaultStatLogLevel)
		api.ConfigStatReportLogger(*conf.LogDir, plog.DefaultStatReportLogLevel)
		api.ConfigNetworkLogger(*conf.LogDir, plog.DefaultNetworkLogLevel)
	}
	if conf.Debug {
		plog.GetBaseLogger().SetLogLevel(plog.DebugLog)
	}
}

// Setup 初始化
func (f *SelectorFactory) Setup(name string, configDec plugin.Decoder) error {
	if configDec == nil {
		return errors.New("selector config decoder empty")
	}
	conf := &Config{}
	err := configDec.Decode(conf)
	if err != nil {
		return err
	}

	// 如果没设置协议默认使用 grpc 协议
	if len(conf.Protocol) == 0 {
		conf.Protocol = "grpc"
	}

	// 初始化日志
	conf.setLog()
	sdkCtx, err := newSDKContext(conf)
	if err != nil {
		return fmt.Errorf("new sdk ctx err: %s", err.Error())
	}
	return setupComponents(sdkCtx, conf)
}

func setupComponents(sdkCtx api.SDKContext, conf *Config) error {
	setDefault := conf.getSetDefault()
	enableServiceRouter := conf.getEnableServiceRouter()
	enableCanary := conf.getEnableCanary()
	if err := discovery.Setup(sdkCtx, &discovery.Config{}, setDefault); err != nil {
		return err
	}

	// 初始化服务路由
	if err := servicerouter.Setup(
		sdkCtx,
		&servicerouter.Config{
			Enable:       enableServiceRouter,
			EnableCanary: enableCanary,
		},
		setDefault,
	); err != nil {
		return err
	}
	if err := setupLoadbalance(sdkCtx, conf, setDefault); err != nil {
		return err
	}
	if err := circuitbreaker.Setup(
		sdkCtx,
		&circuitbreaker.Config{
			ReportTimeout: conf.ReportTimeout,
		},
		setDefault,
	); err != nil {
		return err
	}
	if err := selector.Setup(sdkCtx,
		&selector.Config{
			Enable:          enableServiceRouter,
			EnableCanary:    enableCanary,
			ReportTimeout:   conf.ReportTimeout,
			EnableTransMeta: conf.EnableTransMeta,
		}); err != nil {
		return err
	}
	return nil
}

func setupLoadbalance(sdkCtx api.SDKContext, conf *Config, setDefault bool) error {
	if len(conf.Loadbalance.Name) == 0 {
		conf.Loadbalance.Name = append(
			conf.Loadbalance.Name,
			loadbalance.LoadBalancerWR,
			loadbalance.LoadBalancerHash,
			loadbalance.LoadBalancerRingHash,
			loadbalance.LoadBalancerL5CST,
			loadbalance.LoadBalancerMaglev,
		)
	}
	for index, balanceType := range conf.Loadbalance.Name {
		// 默认设置北极星为寻址方式的前提下，第一个负载均衡则设置为默认的负载均衡方式
		isDefault := setDefault && index == 0
		if err := loadbalance.Setup(sdkCtx, balanceType, isDefault); err != nil {
			return err
		}
	}
	return nil
}

func setSdkCircuitBreaker(c *config.ConfigurationImpl, cfg *Config) {
	if len(cfg.CircuitBreaker.Chain) > 0 {
		c.Consumer.CircuitBreaker.Chain = cfg.CircuitBreaker.Chain
	}
	if cfg.CircuitBreaker.CheckPeriod != nil {
		c.Consumer.CircuitBreaker.CheckPeriod = cfg.CircuitBreaker.CheckPeriod
	}
	if cfg.CircuitBreaker.RequestCountAfterHalfOpen != nil {
		c.Consumer.CircuitBreaker.RequestCountAfterHalfOpen = *cfg.CircuitBreaker.RequestCountAfterHalfOpen
	}
	if cfg.CircuitBreaker.SleepWindow != nil {
		c.Consumer.CircuitBreaker.SleepWindow = cfg.CircuitBreaker.SleepWindow
	}
	if cfg.CircuitBreaker.SuccessCountAfterHalfOpen != nil {
		c.Consumer.CircuitBreaker.SuccessCountAfterHalfOpen = *cfg.CircuitBreaker.SuccessCountAfterHalfOpen
	}
	setErrorCount(c, cfg)
	setErrorRate(c, cfg)
}

func setErrorCount(c *config.ConfigurationImpl, cfg *Config) {
	if cfg.CircuitBreaker.ErrorCount == nil {
		return
	}
	config := &errorcount.Config{}
	config.SetDefault()
	errorCount := cfg.CircuitBreaker.ErrorCount
	if errorCount.ContinuousErrorThreshold != nil {
		config.ContinuousErrorThreshold = *errorCount.ContinuousErrorThreshold
	}
	if errorCount.MetricNumBuckets != nil {
		config.MetricNumBuckets = *errorCount.MetricNumBuckets
	}
	if errorCount.MetricStatTimeWindow != nil {
		config.MetricStatTimeWindow = errorCount.MetricStatTimeWindow
	}
	c.Consumer.CircuitBreaker.SetPluginConfig("errorCount", config)
}

func setErrorRate(c *config.ConfigurationImpl, cfg *Config) {
	if cfg.CircuitBreaker.ErrorRate == nil {
		return
	}
	config := &errorrate.Config{}
	config.SetDefault()
	errorRate := cfg.CircuitBreaker.ErrorRate
	if errorRate.ErrorRateThreshold != nil {
		config.ErrorRateThreshold = *errorRate.ErrorRateThreshold
	}
	if errorRate.MetricNumBuckets != nil {
		config.MetricNumBuckets = *errorRate.MetricNumBuckets
	}
	if errorRate.RequestVolumeThreshold != nil {
		config.RequestVolumeThreshold = *errorRate.RequestVolumeThreshold
	}
	if errorRate.MetricStatTimeWindow != nil {
		config.MetricStatTimeWindow = errorRate.MetricStatTimeWindow
	}
	c.Consumer.CircuitBreaker.SetPluginConfig("errorRate", config)
}

func setSdkOutlierDetection(c *config.ConfigurationImpl, cfg *Config) {
	if cfg.OutlierDetection.Enable != nil {
		c.Consumer.OutlierDetection.Enable = cfg.OutlierDetection.Enable
	}
	if cfg.OutlierDetection.CheckPeriod != nil {
		c.Consumer.OutlierDetection.CheckPeriod = cfg.OutlierDetection.CheckPeriod
	}
	// 只开启tcp连接探测
	c.Consumer.OutlierDetection.Chain = []string{"tcp"}
}

func setSdkProperty(c *config.ConfigurationImpl, cfg *Config) {
	if cfg.Timeout != 0 {
		timeout := time.Duration(cfg.Timeout) * time.Millisecond
		c.Global.API.Timeout = &timeout
		// 如果设置了超时则需要把最大重试次数设置为 0
		c.Global.API.MaxRetryTimes = 0
	}
	if cfg.Discovery.RefreshInterval != 0 {
		refreshInterval := time.Duration(cfg.Discovery.RefreshInterval) * time.Millisecond
		c.Consumer.LocalCache.ServiceRefreshInterval = &refreshInterval
	}
	// 设置服务缓存持久化目录
	if cfg.PersistDir != nil {
		c.Consumer.LocalCache.PersistDir = *cfg.PersistDir
	}
	// 设置sdk缓存保持时间
	if cfg.ServiceExpireTime != nil {
		c.GetConsumer().GetLocalCache().SetServiceExpireTime(*cfg.ServiceExpireTime)
	}
	if cfg.ClusterService.Discover != "" {
		c.Global.GetSystem().GetDiscoverCluster().SetService(cfg.ClusterService.Discover)
	}
	if cfg.ClusterService.HealthCheck != "" {
		c.Global.GetSystem().GetHealthCheckCluster().SetService(cfg.ClusterService.HealthCheck)
	}
	if cfg.ClusterService.Monitor != "" {
		c.Global.GetSystem().GetMonitorCluster().SetService(cfg.ClusterService.Monitor)
	}
	// 设置服务路由
	if cfg.ServiceRouter.NearbyMatchLevel != "" {
		c.Consumer.ServiceRouter.GetNearbyConfig().SetMatchLevel(cfg.ServiceRouter.NearbyMatchLevel)
	}
}

func newSDKContext(cfg *Config) (api.SDKContext, error) {
	var c *config.ConfigurationImpl
	cfg.AddressList = strings.TrimSpace(cfg.AddressList)
	if len(cfg.AddressList) > 0 {
		addressList := strings.Split(cfg.AddressList, ",")
		c = config.NewDefaultConfiguration(addressList)
	} else {
		c = config.NewDefaultConfigurationWithDomain()
	}

	// 设置 joinPoint，会覆盖 address_list 的配置
	if cfg.JoinPoint != nil {
		c.GetGlobal().GetServerConnector().SetJoinPoint(*cfg.JoinPoint)
	}
	c.Global.ServerConnector.Protocol = cfg.Protocol
	connectTimeout := selector.DefaultConnectTimeout
	if cfg.ConnectTimeout != 0 {
		connectTimeout = time.Millisecond * time.Duration(cfg.ConnectTimeout)
	}
	c.Global.ServerConnector.ConnectTimeout = model.ToDurationPtr(connectTimeout)
	messageTimeout := selector.DefaultMessageTimeout
	if cfg.MessageTimeout != nil {
		messageTimeout = *cfg.MessageTimeout
	}

	// 增加按照被调服务env过滤插件
	c.Consumer.ServiceRouter.Chain = append([]string{config.DefaultServiceRouterDstMeta},
		c.Consumer.ServiceRouter.Chain...)

	// 增加金丝雀路由chain
	if cfg.EnableCanary != nil && *cfg.EnableCanary {
		c.Consumer.ServiceRouter.Chain = append(c.Consumer.ServiceRouter.Chain,
			config.DefaultServiceRouterCanary)
	}

	c.GetGlobal().GetServerConnector().SetMessageTimeout(messageTimeout)
	// 配置熔断策略
	setSdkCircuitBreaker(c, cfg)
	// 配置熔断探活策略
	setSdkOutlierDetection(c, cfg)
	// 配置其他属性
	setSdkProperty(c, cfg)
	sdkCtx, err := api.InitContextByConfig(c)
	if err != nil {
		return nil, err
	}
	return sdkCtx, nil
}
