// Package dsn 数据层存储寻址方式 data source name，主要用于mysql mongodb等database寻址
package dsn

import (
	"errors"
	"sync"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/naming/selector"
)

// SeletorName dsn selector name
var SeletorName string = "dsn"

func init() {
	selector.Register(SeletorName, DefaultSelector)
}

// DefaultSelector dsn default selector
var DefaultSelector = &DsnSelector{dsns: make(map[string]*registry.Node)}

// DsnSelector 返回原始service name节点，内部做了内存缓存
type DsnSelector struct {
	dsns map[string]*registry.Node
	lock sync.RWMutex
}

// Select select address from dsn://user:passwd@tcp(ip:port)/db
func (s *DsnSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {

	if len(serviceName) == 0 {
		return nil, errors.New("dsn address can not be empty")
	}

	s.lock.RLock()
	node, ok := s.dsns[serviceName]
	s.lock.RUnlock()

	if ok {
		return node, nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	node, ok = s.dsns[serviceName]
	if ok {
		return node, nil
	}

	node = &registry.Node{
		ServiceName: serviceName,
		Address:     serviceName,
	}

	s.dsns[serviceName] = node
	return node, nil
}

// Report dsn selector no need to report
func (s *DsnSelector) Report(node *registry.Node, cost time.Duration, err error) error {
	return nil
}
