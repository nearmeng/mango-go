package loadbalance

import (
	"fmt"
	"net"
	"strconv"

	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/config"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	"git.code.oa.com/polaris/polaris-go/pkg/plugin/common"
	"git.code.oa.com/polaris/polaris-go/pkg/plugin/loadbalancer"
	"git.code.oa.com/trpc-go/trpc-go/naming/loadbalance"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
)

const (
	// LoadBalancerWR 加权轮训
	LoadBalancerWR = "polaris_wr"
	// LoadBalancerHash 哈希
	LoadBalancerHash = "polaris_hash"
	// LoadBalancerRingHash 环哈希
	LoadBalancerRingHash = "polaris_ring_hash"
	// LoadBalancerMaglev maglev
	LoadBalancerMaglev = "polaris_maglev"
	// LoadBalancerL5CST l5cst
	LoadBalancerL5CST = "polaris_l5cst"
)

var loadBalanceMap map[string]string = map[string]string{
	LoadBalancerWR:       config.DefaultLoadBalancerWR,
	LoadBalancerHash:     config.DefaultLoadBalancerHash,
	LoadBalancerRingHash: config.DefaultLoadBalancerRingHash,
	LoadBalancerMaglev:   config.DefaultLoadBalancerMaglev,
	LoadBalancerL5CST:    config.DefaultLoadBalancerL5CST,
}

const (
	setEnableKey   string = "internal-enable-set"
	setNameKey     string = "internal-set-name"
	setEnableValue string = "Y"
	containerKey   string = "container_name"
)

// Setup 注册
func Setup(sdkCtx api.SDKContext, balanceType string, setDefault bool) error {
	name, ok := loadBalanceMap[balanceType]
	if !ok {
		return fmt.Errorf("loadbalance %s not implement", balanceType)
	}

	lb, err := New(sdkCtx, name)
	if err != nil {
		return err
	}

	loadbalance.Register(balanceType, lb)
	if setDefault {
		loadbalance.SetDefaultLoadBalancer(lb)
	}
	return nil
}

// New 新建
func New(sdkCtx api.SDKContext, name string) (*WRLoadBalancer, error) {

	loadBalancer, err := sdkCtx.GetPlugins().GetPlugin(common.TypeLoadBalancer, name)
	if err != nil {
		return nil, err
	}
	lb := loadBalancer.(loadbalancer.LoadBalancer)

	return &WRLoadBalancer{
		sdkCtx: sdkCtx,
		lb:     lb,
	}, nil
}

// WRLoadBalancer 负载均衡对象
type WRLoadBalancer struct {
	sdkCtx api.SDKContext
	lb     loadbalancer.LoadBalancer
}

// Select 选择负载均衡节点
func (wr *WRLoadBalancer) Select(serviceName string,
	list []*registry.Node, opt ...loadbalance.Option) (*registry.Node, error) {
	opts := &loadbalance.Options{}
	for _, o := range opt {
		o(opts)
	}

	if len(list) == 0 {
		return nil, loadbalance.ErrNoServerAvailable
	}
	cluster := list[0].Metadata["cluster"].(*model.Cluster)
	serviceInstances := list[0].Metadata["serviceInstances"].(model.ServiceInstances)
	envKey := list[0].EnvKey

	criteria := &loadbalancer.Criteria{
		Cluster: cluster,
		HashKey: []byte(opts.Key),
	}
	inst, err := loadbalancer.ChooseInstance(wr.sdkCtx.GetValueContext(), wr.lb, criteria, serviceInstances)
	if err != nil {
		return nil, fmt.Errorf("choose instance err: %s", err.Error())
	}
	var (
		setName       string
		containerName string
	)
	if inst.GetMetadata() != nil {
		containerName = inst.GetMetadata()[containerKey]
		if enable := inst.GetMetadata()[setEnableKey]; enable == setEnableValue {
			setName = inst.GetMetadata()[setNameKey]
		}
	}

	node := &registry.Node{
		ContainerName: containerName,
		SetName:       setName,
		ServiceName:   serviceName,
		Address:       net.JoinHostPort(inst.GetHost(), strconv.Itoa(int(inst.GetPort()))),
		Weight:        inst.GetWeight(),
		EnvKey:        envKey,
		Metadata: map[string]interface{}{
			"instance": inst,
		},
	}

	return node, nil
}
