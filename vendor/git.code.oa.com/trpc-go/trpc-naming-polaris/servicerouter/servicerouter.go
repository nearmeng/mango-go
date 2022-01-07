package servicerouter

import (
	"errors"
	"fmt"
	"strings"

	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/config"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	"git.code.oa.com/polaris/polaris-go/pkg/model/pb"
	namingpb "git.code.oa.com/polaris/polaris-go/pkg/model/pb/v1"
	"git.code.oa.com/polaris/polaris-go/pkg/plugin/common"
	"git.code.oa.com/polaris/polaris-go/pkg/plugin/servicerouter"
	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	tsr "git.code.oa.com/trpc-go/trpc-go/naming/servicerouter"
	"github.com/golang/protobuf/ptypes/wrappers"
)

var CanaryKey string = "trpc-canary"

// Setup 注册
func Setup(sdkCtx api.SDKContext, cfg *Config, setDefault bool) error {
	s := &ServiceRouter{
		consumer: api.NewConsumerAPIByContext(sdkCtx),
		cfg:      cfg,
		sdkCtx:   sdkCtx,
	}

	// 初始化规则路由
	ruleBased, err := sdkCtx.GetPlugins().GetPlugin(
		common.TypeServiceRouter, config.DefaultServiceRouterRuleBased)
	if err != nil {
		log.Errorf("get service router plugin err: %s\n", err.Error())
		return err
	}
	s.RuleBased = ruleBased.(servicerouter.ServiceRouter)

	// 初始化就近路由
	nearbyBased, err := sdkCtx.GetPlugins().GetPlugin(
		common.TypeServiceRouter, config.DefaultServiceRouterNearbyBased)
	if err != nil {
		log.Errorf("get service router plugin err: %s\n", err.Error())
		return err
	}
	s.NearbyBased = nearbyBased.(servicerouter.ServiceRouter)

	//初始化set分组路由
	setDivison, err := sdkCtx.GetPlugins().GetPlugin(
		common.TypeServiceRouter, config.DefaultServiceRouterSetDivision)
	if err != nil {
		log.Errorf("get service router plugin err: %s\n", err.Error())
		return err
	}
	s.SetDivison = setDivison.(servicerouter.ServiceRouter)

	// 初始化过滤不健康节点的路由
	filterOnly, err := sdkCtx.GetPlugins().GetPlugin(
		common.TypeServiceRouter, config.DefaultServiceRouterFilterOnly)
	if err != nil {
		log.Errorf("get service router plugin err: %s\n", err.Error())
		return err
	}
	s.FilterOnly = filterOnly.(servicerouter.ServiceRouter)

	// 初始化按照 meta 过滤的路由
	dstMeta, err := sdkCtx.GetPlugins().GetPlugin(
		common.TypeServiceRouter, config.DefaultServiceRouterDstMeta)
	if err != nil {
		log.Errorf("get service router plugin err: %s\n", err.Error())
		return err
	}
	s.DstMeta = dstMeta.(servicerouter.ServiceRouter)

	// 初始化金丝雀路由插件
	canary, err := sdkCtx.GetPlugins().GetPlugin(
		common.TypeServiceRouter, config.DefaultServiceRouterCanary)
	if err != nil {
		log.Errorf("get service router plugin err: %s\n", err.Error())
		return err
	}
	s.Canary = canary.(servicerouter.ServiceRouter)

	tsr.Register("polaris", s)
	if setDefault {
		tsr.SetDefaultServiceRouter(s)
	}
	return nil
}

// ServiceRouter 服务路由
type ServiceRouter struct {
	sdkCtx      api.SDKContext
	consumer    api.ConsumerAPI
	RuleBased   servicerouter.ServiceRouter
	NearbyBased servicerouter.ServiceRouter
	FilterOnly  servicerouter.ServiceRouter
	DstMeta     servicerouter.ServiceRouter
	SetDivison  servicerouter.ServiceRouter
	Canary      servicerouter.ServiceRouter
	cfg         *Config
}

func hasEnv(r *namingpb.Route, env string) bool {
	var hasEnv bool
	for _, source := range r.GetSources() {
		if source.GetMetadata() == nil {
			continue
		}
		value, ok := source.GetMetadata()["env"]
		if !ok {
			continue
		}
		if value.GetValue().GetValue() == env {
			hasEnv = true
			break
		}
	}

	return hasEnv
}

func getDestination(r *namingpb.Route) []string {
	var result []string
	for _, dest := range r.GetDestinations() {
		if dest.GetMetadata() == nil {
			continue
		}
		value, ok := dest.GetMetadata()["env"]
		if !ok {
			continue
		}
		if dest.GetService().GetValue() == "*" && value.GetValue().GetValue() != "" {
			result = append(result, value.GetValue().GetValue())
		}
	}
	return result
}

// 从服务路由里面找出环境优先级信息
func getEnvPriority(routes []*namingpb.Route, env string) string {
	result := []string{}
	for _, r := range routes {
		if !hasEnv(r, env) {
			continue
		}
		dest := getDestination(r)
		result = append(result, dest...)
	}
	return strings.Join(result, ",")
}

// 获取服务的出规则路由
func getOutboundsRoute(rules *model.ServiceRuleResponse) []*namingpb.Route {
	if rules != nil && rules.GetType() == model.EventRouting {
		value, ok := rules.GetValue().(*namingpb.Routing)
		if ok {
			return value.GetOutbounds()
		}
	}
	return []*namingpb.Route{}
}

func (s *ServiceRouter) setEnable(
	srcServiceInfo *model.ServiceInfo,
	dstServiceInfo *model.ServiceInfo,
	opts *tsr.Options,
	chain []servicerouter.ServiceRouter,
) []servicerouter.ServiceRouter {
	sourceSetName := opts.SourceSetName
	dstSetName := opts.DestinationSetName
	if len(sourceSetName) != 0 || len(dstSetName) != 0 {
		//启用了set分组
		if len(sourceSetName) != 0 {
			if srcServiceInfo.Metadata == nil {
				srcServiceInfo.Metadata = map[string]string{
					setEnableKey: setEnableValue,
					setNameKey:   sourceSetName,
				}
			}
			srcServiceInfo.Metadata[setEnableKey] = setEnableValue
			srcServiceInfo.Metadata[setNameKey] = sourceSetName
		}
		if len(dstSetName) != 0 {
			if dstServiceInfo.Metadata == nil {
				dstServiceInfo.Metadata = map[string]string{
					setEnableKey: setEnableValue,
					setNameKey:   dstSetName,
				}
			}
			dstServiceInfo.Metadata[setEnableKey] = setEnableValue
			dstServiceInfo.Metadata[setNameKey] = dstSetName
		}
		chain = append(chain, s.SetDivison)
	}
	return chain
}

func (s *ServiceRouter) filterWithEnv(
	serviceInstances model.ServiceInstances,
	sourceService, destService *model.ServiceInfo, opts *tsr.Options) ([]*registry.Node, error) {
	envList := []string{}
	if len(opts.EnvTransfer) > 0 {
		envList = strings.Split(opts.EnvTransfer, ",")
	}

	sourceService.Metadata = map[string]string{
		"env": opts.SourceEnvName,
	}
	canaryValue := getCanaryValue(opts)
	routeRules := buildRouteRules(opts.SourceNamespace,
		opts.SourceServiceName, opts.SourceEnvName, opts.Namespace, envList)
	routeInfo := &servicerouter.RouteInfo{
		SourceService:    sourceService,
		SourceRouteRule:  routeRules,
		DestService:      destService,
		FilterOnlyRouter: s.FilterOnly,
		Canary:           canaryValue,
	}

	// 考虑set分组情况
	chain := []servicerouter.ServiceRouter{s.RuleBased}
	chain = s.setEnable(sourceService, destService, opts, chain)
	chain = append(chain, s.NearbyBased)
	if s.cfg.EnableCanary {
		chain = append(chain, s.Canary)
	}
	instances, cluster, _, err := servicerouter.GetFilterInstances(s.sdkCtx.GetValueContext(),
		chain, routeInfo, serviceInstances)
	if err != nil {
		return nil, fmt.Errorf("filter instance with env err: %s", err.Error())
	}

	return instanceToNode(instances, opts.EnvTransfer, cluster, serviceInstances), nil
}

func (s *ServiceRouter) filter(
	serviceInstances model.ServiceInstances,
	sourceService, destService *model.ServiceInfo, opts *tsr.Options) ([]*registry.Node, error) {

	sourceRouteRules, err := s.consumer.GetRouteRule(&api.GetServiceRuleRequest{
		GetServiceRuleRequest: model.GetServiceRuleRequest{
			Namespace: sourceService.Namespace,
			Service:   sourceService.Service,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get source service ns: %s, service: %s route rule err: %s",
			sourceService.Namespace, sourceService.Service, err.Error())
	}
	canaryValue := getCanaryValue(opts)

	// 则先判断自身是否有出规则
	// 没有出规则则直接跳过服务路由，只过滤不健康的节点
	// 否则则使用本节点的env和key进行自身出规则的筛选
	chain := []servicerouter.ServiceRouter{}
	var newEnvStr string
	outbounds := getOutboundsRoute(sourceRouteRules)
	if len(outbounds) == 0 {
		chain = s.setEnable(sourceService, destService, opts, chain)
		chain = append(chain, s.NearbyBased)
		if s.cfg.EnableCanary {
			chain = append(chain, s.Canary)
		}
	} else {
		// 主调服务元数据，用于规则路由
		sourceService.Metadata = make(map[string]string)
		for key, value := range opts.SourceMetadata {
			if len(key) > 0 && len(value) > 0 {
				sourceService.Metadata[key] = value
			}
		}

		// 配置环境路由，如果设置环境key，优先使用环境key
		if len(opts.EnvKey) > 0 {
			sourceService.Metadata["key"] = opts.EnvKey
		} else {
			sourceService.Metadata["env"] = opts.SourceEnvName
		}
		newEnvStr = getEnvPriority(outbounds, opts.SourceEnvName)

		chain = append(chain, s.RuleBased)
		chain = s.setEnable(sourceService, destService, opts, chain)
		chain = append(chain, s.NearbyBased)
		if s.cfg.EnableCanary {
			chain = append(chain, s.Canary)
		}
	}

	routeInfo := &servicerouter.RouteInfo{
		SourceService:    sourceService,
		SourceRouteRule:  sourceRouteRules,
		DestService:      destService,
		FilterOnlyRouter: s.FilterOnly,
		Canary:           canaryValue,
	}
	instances, cluster, _, err := servicerouter.GetFilterInstances(
		s.sdkCtx.GetValueContext(),
		chain,
		routeInfo,
		serviceInstances,
	)
	if err != nil {
		return nil, fmt.Errorf("filter instances without transfer env err: %s", err.Error())
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("env %s do not have instances, key: %s",
			opts.SourceEnvName, opts.EnvKey)
	}

	return instanceToNode(instances, newEnvStr, cluster, serviceInstances), nil
}

func (s *ServiceRouter) filterWithoutServiceRouter(
	serviceInstances model.ServiceInstances,
	sourceService, destService *model.ServiceInfo, opts *tsr.Options) ([]*registry.Node, error) {
	chain := []servicerouter.ServiceRouter{}
	if len(opts.DestinationEnvName) > 0 {
		chain = append(chain, s.DstMeta)
		destService.Metadata = map[string]string{
			"env": opts.DestinationEnvName,
		}
	}
	canaryValue := getCanaryValue(opts)
	chain = s.setEnable(sourceService, destService, opts, chain)
	chain = append(chain, s.NearbyBased)
	if s.cfg.EnableCanary {
		chain = append(chain, s.Canary)
	}
	routeInfo := &servicerouter.RouteInfo{
		SourceService:    sourceService,
		DestService:      destService,
		FilterOnlyRouter: s.FilterOnly,
		Canary:           canaryValue,
	}
	instances, cluster, _, err := servicerouter.GetFilterInstances(
		s.sdkCtx.GetValueContext(), chain, routeInfo, serviceInstances)
	if err != nil {
		return nil, fmt.Errorf("filter instances err: %s", err.Error())
	}
	if len(instances) == 0 {
		return nil, errors.New("filter instances no instances available")
	}
	return instanceToNode(instances, "", cluster, serviceInstances), nil
}

// Filter 根据路由规则过滤实例
func (s *ServiceRouter) Filter(serviceName string,
	nodes []*registry.Node, opt ...tsr.Option) ([]*registry.Node, error) {
	if len(nodes) == 0 {
		return nil, errors.New("servicerouter: no node available")
	}
	serviceInstances, ok := nodes[0].Metadata["service_instances"].(model.ServiceInstances)
	if !ok {
		return nil, errors.New("sercice instances invalid")
	}

	opts := &tsr.Options{}
	for _, o := range opt {
		o(opts)
	}
	log.Tracef("[NAMING-POLARIS] servicerouter options: %+v", opts)
	sourceService := &model.ServiceInfo{
		Service:   opts.SourceServiceName,
		Namespace: opts.SourceNamespace,
	}
	destService := &model.ServiceInfo{
		Service:   serviceName,
		Namespace: opts.Namespace,
	}

	// 主调服务信息不存在则不走服务路由
	if len(sourceService.Service) == 0 ||
		len(sourceService.Namespace) == 0 ||
		opts.DisableServiceRouter ||
		!s.cfg.Enable {
		return s.filterWithoutServiceRouter(serviceInstances, sourceService, destService, opts)
	}

	// 如果没有透传环境信息
	if len(opts.EnvTransfer) == 0 {
		return s.filter(serviceInstances, sourceService, destService, opts)
	}
	return s.filterWithEnv(serviceInstances, sourceService, destService, opts)
}

// 根据透传的环境优先级列表构建查询规则
func buildRouteRules(sourceNamespace, sourceServiceName,
	sourceEnv, destNamespace string, envList []string) model.ServiceRule {
	route := &namingpb.Route{
		Sources: []*namingpb.Source{
			{
				Namespace: &wrappers.StringValue{
					Value: sourceNamespace,
				},
				Service: &wrappers.StringValue{
					Value: sourceServiceName,
				},
				Metadata: map[string]*namingpb.MatchString{
					"env": {
						Type: namingpb.MatchString_EXACT,
						Value: &wrappers.StringValue{
							Value: sourceEnv,
						},
					},
				},
			},
		},
	}
	dests := []*namingpb.Destination{}
	for i, env := range envList {
		dest := &namingpb.Destination{
			Namespace: &wrappers.StringValue{
				Value: destNamespace,
			},
			Service: &wrappers.StringValue{
				Value: "*",
			},
			Priority: &wrappers.UInt32Value{
				Value: uint32(i),
			},
			Weight: &wrappers.UInt32Value{
				Value: 100,
			},
			Metadata: map[string]*namingpb.MatchString{
				"env": {
					Type: namingpb.MatchString_EXACT,
					Value: &wrappers.StringValue{
						Value: env,
					},
				},
			},
		}
		dests = append(dests, dest)
	}

	route.Destinations = dests
	value := &namingpb.Routing{
		Namespace: &wrappers.StringValue{
			Value: sourceNamespace,
		},
		Service: &wrappers.StringValue{
			Value: sourceServiceName,
		},
		Outbounds: []*namingpb.Route{route},
	}

	rule := &namingpb.DiscoverResponse{
		Code: &wrappers.UInt32Value{Value: namingpb.ExecuteSuccess},
		Info: &wrappers.StringValue{Value: "create from local"},
		Type: namingpb.DiscoverResponse_ROUTING,
		Service: &namingpb.Service{
			Name:      &wrappers.StringValue{Value: sourceServiceName},
			Namespace: &wrappers.StringValue{Value: sourceNamespace},
		},
		Instances: nil,
		Routing:   value,
	}
	return pb.NewRoutingRuleInProto(rule)
}

func instanceToNode(instances []model.Instance,
	env string, cluster *model.Cluster, resp model.ServiceInstances) []*registry.Node {
	list := make([]*registry.Node, 0, len(instances))
	for range instances {
		n := &registry.Node{
			EnvKey: env,
			Metadata: map[string]interface{}{
				"serviceInstances": resp,
				"cluster":          cluster,
			},
		}
		list = append(list, n)
		// 节点列表对于框架来说无意义，为了性能考虑，只保留一个用来存储 sdk 的节点列表信息
		break
	}
	return list
}

func getCanaryValue(opts *tsr.Options) string {
	if opts.Ctx == nil {
		return ""
	}
	ctx := opts.Ctx
	msg := codec.Message(ctx)
	metaData := msg.ClientMetaData()
	if metaData == nil {
		return ""
	}
	return string(metaData[CanaryKey])
}

// WithCanary 设置金丝雀 metadata
func WithCanary(val string) client.Option {
	return func(o *client.Options) {
		client.WithMetaData(CanaryKey, []byte(val))(o)
	}
}
