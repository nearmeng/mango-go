package selector

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/config"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/naming/selector"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/circuitbreaker"
	"git.code.oa.com/trpc-go/trpc-naming-polaris/servicerouter"
)

const (
	// DefaultConnectTimeout 默认连接超时时间
	DefaultConnectTimeout = time.Second
	// DefaultMessageTimeout 默认消息超时时间
	DefaultMessageTimeout = time.Second
)

var once = &sync.Once{}

// Setup 初始化
func Setup(sdkCtx api.SDKContext, cfg *Config) error {
	s := &Selector{
		consumer: api.NewConsumerAPIByContext(sdkCtx),
		cfg:      cfg,
	}
	selector.Register("polaris", s)
	return nil
}

// RegisterDefault 注册默认 selector
func RegisterDefault() {
	once.Do(func() {
		s, err := New(&Config{
			Protocol:   "grpc",
			Enable:     true,
			UseBuildin: true,
		})
		if err != nil {
			panic(err)
		}
		selector.Register("polaris", s)
	})
}

// Register 根据参数注册selector
func Register(cfg *Config) {
	once.Do(func() {
		s, err := New(cfg)
		if err != nil {
			panic(err)
		}
		selector.Register("polaris", s)
	})
}

// New 新建实例
func New(cfg *Config) (*Selector, error) {
	var c *config.ConfigurationImpl
	if cfg.UseBuildin {
		c = config.NewDefaultConfigurationWithDomain()
	} else {
		c = config.NewDefaultConfiguration(cfg.ServerAddrs)
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "grpc"
	}
	c.Global.ServerConnector.Protocol = cfg.Protocol
	if cfg.RefreshInterval != 0 {
		refreshInterval := time.Duration(cfg.RefreshInterval) * time.Millisecond
		c.Consumer.LocalCache.ServiceRefreshInterval = &refreshInterval
	}
	if cfg.Timeout != 0 {
		timeout := time.Duration(cfg.Timeout) * time.Millisecond
		c.Global.API.Timeout = &timeout
		// 如果设置了超时则需要把最大重试次数设置为 0
		c.Global.API.MaxRetryTimes = 0
	}
	connectTimeout := DefaultConnectTimeout
	if cfg.ConnectTimeout != 0 {
		connectTimeout = time.Millisecond * time.Duration(cfg.ConnectTimeout)
	}
	c.Global.ServerConnector.ConnectTimeout = model.ToDurationPtr(connectTimeout)

	// 增加按照被调服务env过滤插件
	c.Consumer.ServiceRouter.Chain = append([]string{config.DefaultServiceRouterDstMeta},
		c.Consumer.ServiceRouter.Chain...)

	// 增加金丝雀路由chain
	if cfg.EnableCanary {
		c.Consumer.ServiceRouter.Chain = append(c.Consumer.ServiceRouter.Chain,
			config.DefaultServiceRouterCanary)
	}
	// 配置本地cache存储地址
	if cfg.LocalCachePersistDir != "" {
		c.Consumer.LocalCache.PersistDir = cfg.LocalCachePersistDir
	}
	sdkCtx, err := api.InitContextByConfig(c)
	if err != nil {
		return nil, err
	}
	return &Selector{
		consumer: api.NewConsumerAPIByContext(sdkCtx),
		cfg:      cfg,
	}, nil
}

// Selector 路由选择器
type Selector struct {
	consumer api.ConsumerAPI
	cfg      *Config
}

func getMetadata(opts *selector.Options, enableTransMeta bool) map[string]string {
	metadata := make(map[string]string)
	if len(opts.SourceEnvName) > 0 {
		metadata["env"] = opts.SourceEnvName
	}
	// 为解决请求透传字段无法传递给北极星用于meta匹配的问题。约定'selector-meta-'前缀的透传字段，摘除前缀后填入meta，给北极星匹配使用
	if enableTransMeta {
		setTransSelectorMeta(opts, metadata)
	}
	for key, value := range opts.SourceMetadata {
		if len(key) > 0 && len(value) > 0 {
			metadata[key] = value
		}
	}
	return metadata
}

func extractSourceServiceRequestInfo(opts *selector.Options, enableTransMeta bool) *model.ServiceInfo {
	if opts.DisableServiceRouter {
		return nil
	}
	metadata := getMetadata(opts, enableTransMeta)
	if len(metadata) == 0 {
		return nil
	}
	// 北极星支持无主调服务信息的场景
	return &model.ServiceInfo{
		Service:   opts.SourceServiceName,
		Namespace: opts.SourceNamespace,
		Metadata:  metadata,
	}
}

func getDestMetadata(opts *selector.Options) map[string]string {
	var destMeta map[string]string
	// 未开启服务路由的情况下支持选择环境
	if opts.DisableServiceRouter {
		if len(opts.DestinationEnvName) > 0 {
			destMeta = map[string]string{
				"env": opts.DestinationEnvName,
			}
		}
	}
	return destMeta
}

// Select 选择服务节点
func (s *Selector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
	opts := &selector.Options{}
	for _, o := range opt {
		o(opts)
	}
	log.Tracef("[NAMING-POLARIS] select options: %+v", opts)
	namespace := opts.Namespace
	var sourceService *model.ServiceInfo

	if s.cfg.Enable {
		sourceService = extractSourceServiceRequestInfo(opts, s.cfg.EnableTransMeta)
	}
	if opts.LoadBalanceType == "" {
		opts.LoadBalanceType = LoadBalanceWR
	}
	name, ok := loadBalanceMap[opts.LoadBalanceType]
	if !ok {
		return nil, fmt.Errorf("not support loadbalanceType: %s", opts.LoadBalanceType)
	}
	destMeta := getDestMetadata(opts)
	instanceReq := model.GetOneInstanceRequest{
		Service:        serviceName,
		Namespace:      namespace,
		SourceService:  sourceService,
		Metadata:       destMeta,
		LbPolicy:       name,
		ReplicateCount: opts.Replicas,
		Canary:         getCanaryValue(opts),
	}
	if opts.Key != "" {
		instanceReq.HashKey = []byte(opts.Key)
	}
	req := &api.GetOneInstanceRequest{
		GetOneInstanceRequest: instanceReq,
	}
	resp, err := s.consumer.GetOneInstance(req)
	if err != nil {
		return nil, fmt.Errorf("get one instance err: %s", err.Error())
	}
	if len(resp.Instances) == 0 {
		return nil, fmt.Errorf("get one instance return empty")
	}
	inst := resp.Instances[0]
	var setName, containerName string
	if inst.GetMetadata() != nil {
		containerName = inst.GetMetadata()[containerKey]
		if enable := inst.GetMetadata()[setEnableKey]; enable == setEnableValue {
			setName = inst.GetMetadata()[setNameKey]
		}
	}
	return &registry.Node{
		ContainerName: containerName,
		SetName:       setName,
		Address:       net.JoinHostPort(inst.GetHost(), strconv.Itoa(int(inst.GetPort()))),
		ServiceName:   serviceName,
		Weight:        inst.GetWeight(),
		Metadata: map[string]interface{}{
			"instance":  inst,
			"service":   serviceName,
			"namespace": namespace,
		},
	}, nil
}

// Report 服务状态上报
func (s *Selector) Report(node *registry.Node, cost time.Duration, err error) error {
	return circuitbreaker.Report(s.consumer, node, s.cfg.ReportTimeout, cost, err)
}

func getCanaryValue(opts *selector.Options) string {
	if opts.Ctx == nil {
		return ""
	}
	ctx := opts.Ctx
	msg := codec.Message(ctx)
	metaData := msg.ClientMetaData()
	if metaData == nil {
		return ""
	}
	return string(metaData[servicerouter.CanaryKey])
}

func setTransSelectorMeta(opts *selector.Options, selectorMeta map[string]string) {
	if opts.Ctx == nil {
		return
	}
	msg := codec.Message(opts.Ctx)
	for k, v := range msg.ServerMetaData() {
		if strings.HasPrefix(k, selectorMetaPrefix) {
			trimmedKey := strings.TrimPrefix(k, selectorMetaPrefix)
			selectorMeta[trimmedKey] = string(v)
		}
	}
}
