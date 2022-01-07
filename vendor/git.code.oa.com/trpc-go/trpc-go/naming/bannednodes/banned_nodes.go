// Package bannednodes 定义了一个并发安全的节点列表，用户可以将它和 context 绑定起来。
package bannednodes

import (
	"context"
	"sync"

	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
)

// ctxKeyBannedNodes 是 context 的 key。
type ctxKeyBannedNodes struct{}

// bannedNodes 是 context 的 value。
type bannedNodes struct {
	mu        sync.Mutex
	nodes     *Nodes
	mandatory bool
}

// NewCtx 创建一个新的 context，并设置对应的 k-v。
func NewCtx(ctx context.Context, mandatory bool) context.Context {
	return context.WithValue(ctx, ctxKeyBannedNodes{}, &bannedNodes{mandatory: mandatory})
}

// FromCtx 获取节点列表和一个表示是否是强制禁用的布尔变量。
// FromCtx 并不会返回 bannedNodes，而是返回一个只读的链表。用户不会感知到内部实现中的锁。
func FromCtx(ctx context.Context) (nodes *Nodes, mandatory bool, ok bool) {
	bannedNodes, ok := ctx.Value(ctxKeyBannedNodes{}).(*bannedNodes)
	if !ok {
		return nil, false, false
	}

	bannedNodes.mu.Lock()
	defer bannedNodes.mu.Unlock()
	return bannedNodes.nodes, bannedNodes.mandatory, true
}

// Add 添加一个新的节点到 ctx 中。
// 如果 ctx 没有设置 k-v，那么什么也不会发生。
func Add(ctx context.Context, nodes ...*registry.Node) {
	bannedNodes, ok := ctx.Value(ctxKeyBannedNodes{}).(*bannedNodes)
	if !ok {
		return
	}

	bannedNodes.mu.Lock()
	defer bannedNodes.mu.Unlock()
	for _, node := range nodes {
		bannedNodes.nodes = &Nodes{
			next: bannedNodes.nodes,
			node: node,
		}
	}
}

// Nodes 是一个 registry.Node 链表。
type Nodes struct {
	next *Nodes
	node *registry.Node
}

// Range 类似于 sync.Map 方法中的 Range（它不是并发安全的）。
// 它会线性地为 Nodes 中的所有 node 调用 f。如果 f 返回失败，Range 会停止迭代。
// 只有完全遍历，Range 才会返回 true，否则，它会返回 false。
// 用户不应该在 f 中更改 n 的信息。
func (nodes *Nodes) Range(f func(n *registry.Node) bool) bool {
	if nodes == nil {
		return true
	}
	if f(nodes.node) {
		return nodes.next.Range(f)
	}
	return false
}
