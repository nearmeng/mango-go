// Package selector client后端路由选择器，通过service name获取一个节点，内部调用服务发现，负载均衡，熔断隔离
package selector

import (
	"errors"
	"strings"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/internal/rand"
	"git.code.oa.com/trpc-go/trpc-go/naming/bannednodes"
	"git.code.oa.com/trpc-go/trpc-go/naming/discovery"
	"git.code.oa.com/trpc-go/trpc-go/naming/loadbalance"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/naming/servicerouter"
)

func init() {
	Register("ip", NewIPSelector())  // ip://ip:port
	Register("dns", NewIPSelector()) // dns://domain:port
}

// ipSelector 基于ip列表的selector
type ipSelector struct {
	safeRand *rand.SafeRand
}

// NewIPSelector 新建一个ip的接收器
func NewIPSelector() *ipSelector {
	return &ipSelector{
		safeRand: rand.NewSafeRand(time.Now().UnixNano()),
	}
}

// Select 默认的ip selector， 输入service name是 ip1:port1,ip2:port2, 支持多ip。
// 如果 ctx 中设置了 bannedNodes，那么 Select 会尽可能地选择其他节点。
func (s *ipSelector) Select(
	serviceName string, opt ...Option,
) (node *registry.Node, err error) {
	if serviceName == "" {
		return nil, errors.New("serviceName empty")
	}

	var o Options = Options{
		DiscoveryOptions:     make([]discovery.Option, 0, defaultDiscoveryOptionsSize),
		ServiceRouterOptions: make([]servicerouter.Option, 0, defaultServiceRouterOptionsSize),
		LoadBalanceOptions:   make([]loadbalance.Option, 0, defaultLoadBalanceOptionsSize),
	}
	for _, opt := range opt {
		opt(&o)
	}
	if o.Ctx == nil {
		addr, err := s.chooseOne(serviceName)
		if err != nil {
			return nil, err
		}
		return &registry.Node{ServiceName: serviceName, Address: addr}, nil
	}

	bans, mandatory, ok := bannednodes.FromCtx(o.Ctx)
	if !ok {
		addr, err := s.chooseOne(serviceName)
		if err != nil {
			return nil, err
		}
		return &registry.Node{ServiceName: serviceName, Address: addr}, nil
	}

	defer func() {
		if err == nil {
			bannednodes.Add(o.Ctx, node)
		}
	}()

	addr, err := s.chooseUnbanned(strings.Split(serviceName, ","), bans)
	if !mandatory && err != nil {
		addr, err = s.chooseOne(serviceName)
	}
	if err != nil {
		return nil, err
	}
	return &registry.Node{ServiceName: serviceName, Address: addr}, nil
}

func (s *ipSelector) chooseOne(serviceName string) (string, error) {
	num := strings.Count(serviceName, ",") + 1
	if num == 1 {
		return serviceName, nil
	}

	var addr string
	r := s.safeRand.Intn(num)
	for i := 0; i <= r; i++ {
		j := strings.IndexByte(serviceName, ',')
		if j < 0 {
			addr = serviceName
			break
		}
		addr, serviceName = serviceName[:j], serviceName[j+1:]
	}
	return addr, nil
}

func (s *ipSelector) chooseUnbanned(addrs []string, bans *bannednodes.Nodes) (string, error) {
	if len(addrs) == 0 {
		return "", errors.New("no available targets")
	}

	r := s.safeRand.Intn(len(addrs))
	if !bans.Range(func(n *registry.Node) bool {
		return n.Address != addrs[r]
	}) {
		return s.chooseUnbanned(append(addrs[:r], addrs[r+1:]...), bans)
	}
	return addrs[r], nil
}

// Report 空实现
func (s *ipSelector) Report(*registry.Node, time.Duration, error) error {
	return nil
}
