package loadbalance

import (
	"time"

	"git.code.oa.com/trpc-go/trpc-go/internal/rand"
	"git.code.oa.com/trpc-go/trpc-go/naming/bannednodes"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
)

func init() {
	Register(LoadBalanceRandom, NewRandom())
}

// Random 随机负载均衡
type Random struct {
	safeRand *rand.SafeRand
}

// 创建一个随机负载的接收器
func NewRandom() *Random {
	return &Random{
		safeRand: rand.NewSafeRand(time.Now().UnixNano()),
	}
}

// Select 随机从列表挑选一个节点。如果 ctx 中设置了 bannedNodes，那么 Select 会尽可能地选择其他节点。
func (b *Random) Select(
	serviceName string,
	nodes []*registry.Node,
	opts ...Option,
) (node *registry.Node, err error) {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}

	if o.Ctx == nil {
		return b.chooseOne(nodes)
	}

	bans, mandatory, ok := bannednodes.FromCtx(o.Ctx)
	if !ok {
		return b.chooseOne(nodes)
	}

	defer func() {
		if err == nil {
			bannednodes.Add(o.Ctx, node)
		}
	}()

	node, err = b.chooseUnbanned(nodes, bans)
	if !mandatory && err == ErrNoServerAvailable {
		return b.chooseOne(nodes)
	}
	return node, err
}

func (b *Random) chooseOne(nodes []*registry.Node) (*registry.Node, error) {
	if len(nodes) == 0 {
		return nil, ErrNoServerAvailable
	}
	return nodes[b.safeRand.Intn(len(nodes))], nil
}

func (b *Random) chooseUnbanned(
	nodes []*registry.Node,
	bans *bannednodes.Nodes,
) (*registry.Node, error) {
	if len(nodes) == 0 {
		return nil, ErrNoServerAvailable
	}
	i := b.safeRand.Intn(len(nodes))
	if !bans.Range(func(n *registry.Node) bool {
		return n.Address != nodes[i].Address
	}) {
		return b.chooseUnbanned(append(nodes[:i], nodes[i+1:]...), bans)
	}
	return nodes[i], nil
}
