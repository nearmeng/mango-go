package discovery

import (
	"fmt"

	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	"git.code.oa.com/trpc-go/trpc-go/naming/discovery"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
)

// Setup 注册
func Setup(sdkCtx api.SDKContext, cfg *Config, setDefault bool) error {
	d := &Discovery{
		consumer: api.NewConsumerAPIByContext(sdkCtx),
		cfg:      cfg,
	}

	discovery.Register("polaris", d)
	if setDefault {
		discovery.SetDefaultDiscovery(d)
	}
	return nil
}

// Discovery 服务发现
type Discovery struct {
	consumer api.ConsumerAPI
	cfg      *Config
}

func checkOpts(serviceName string, opt ...discovery.Option) (*discovery.Options, error) {
	opts := &discovery.Options{}

	for _, o := range opt {
		o(opts)
	}

	if len(serviceName) == 0 || len(opts.Namespace) == 0 {
		return nil, fmt.Errorf("service or namespace is empty, namespace: %s, service: %s",
			opts.Namespace, serviceName)
	}

	return opts, nil
}

// List 获取服务的实例列表
func (d *Discovery) List(serviceName string, opt ...discovery.Option) (nodes []*registry.Node, err error) {

	opts, err := checkOpts(serviceName, opt...)
	if err != nil {
		return nil, err
	}

	req := &api.GetInstancesRequest{
		GetInstancesRequest: model.GetInstancesRequest{
			Namespace:                    opts.Namespace,
			Service:                      serviceName,
			IncludeCircuitBreakInstances: true,
			IncludeUnhealthyInstances:    true,
			SkipRouteFilter:              true,
		},
	}
	resp, err := d.consumer.GetInstances(req)
	if err != nil {
		return nil, fmt.Errorf("fail to get instances, err is %s", err.Error())
	}

	list := []*registry.Node{}
	for range resp.Instances {
		n := &registry.Node{
			Metadata: map[string]interface{}{
				"service_instances": resp,
			},
		}
		list = append(list, n)
		// 节点列表对于框架来说无意义，为了性能考虑，只保留一个用来存储 sdk 的节点列表信息
		break
	}

	return list, nil
}
