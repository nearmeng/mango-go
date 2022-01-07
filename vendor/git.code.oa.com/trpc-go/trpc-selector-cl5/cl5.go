package cl5

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.code.oa.com/going/l5"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/naming/selector"
)

func init() {
	selector.Register("cl5", &cl5Selector{})
}

type cl5Selector struct{}

// Select CL5通过service name获取一个后端节点
func (s *cl5Selector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {

	modid, cmdid, err := parse(serviceName)
	if err != nil {
		return nil, err
	}

	addr, err := l5.ApiGetRoute(int32(modid), int32(cmdid))
	if err != nil {
		err = fmt.Errorf("cl5 ApiGetRoute %v", err)
		return nil, err
	}

	node := &registry.Node{
		Address: fmt.Sprint(addr.Ip(), ":", addr.Port()),
		Metadata: map[string]interface{}{
			"cl5_server": addr,
		},
	}
	return node, nil
}

// Report CL5上报当前请求成功或失败
func (s *cl5Selector) Report(node *registry.Node, cost time.Duration, success error) error {
	if node.Metadata == nil {
		return errors.New("cl5: update report metadata nil")
	}

	addr, ok := node.Metadata["cl5_server"]
	if !ok {
		return errors.New("cl5: update report metadata not found key cl5_server")
	}

	server, ok := addr.(*l5.Server)
	if !ok {
		return errors.New("cl5: update report metadata cl5_server type invalid")
	}

	var result int32
	if success != nil {
		result = -1
	}

	err := l5.ApiRouteResultUpdate(server, result, uint64(cost.Nanoseconds()/1e6))
	if err != nil {
		err = fmt.Errorf("cl5 ApiRouteResultUpdate %v", err)
	}
	return err
}

func parse(serviceName string) (modid int, cmdid int, err error) {
	ids := strings.Split(serviceName, ":")
	if len(ids) != 2 {
		err = errors.New("serviceName invalid cl5 format")
		return
	}
	modid, err = strconv.Atoi(ids[0])
	cmdid, err = strconv.Atoi(ids[1])
	return
}
