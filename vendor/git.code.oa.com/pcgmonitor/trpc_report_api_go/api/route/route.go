// Package route 路由
package route

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"git.code.oa.com/polaris/polaris-go/api"
	"git.code.oa.com/polaris/polaris-go/pkg/config"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
)

// Router 路由
type Router interface {
	GetAddr(string, string) (string, error)
}

// Polaris 北极星
type Polaris struct {
	consumer api.ConsumerAPI // 北极星实例
	initSucc bool            // 初始化是否成功
}

// Init 北极星初始化
func (p *Polaris) Init(adds []string, proto string) error {
	var c config.Configuration

	if len(adds) == 0 || proto == "" {
		c = api.NewConfiguration()
	} else {
		c = config.NewDefaultConfiguration(adds)
		c.GetGlobal().GetServerConnector().SetProtocol(proto)
	}

	sdkCtx, err := api.InitContextByConfig(c)
	if err != nil {
		log.Printf("trpc_report_api_go:Polaris init ctx error:%v", err)
		return err
	}

	p.consumer = api.NewConsumerAPIByContext(sdkCtx)
	p.initSucc = true
	return nil
}

// GetAddr 获取地址
func (p *Polaris) GetAddr(namespace, serviceName string) (string, error) {
	if !p.initSucc {
		return "", fmt.Errorf("polaris init fail")
	}

	req := &api.GetOneInstanceRequest{
		GetOneInstanceRequest: model.GetOneInstanceRequest{
			Namespace: namespace,
			Service:   serviceName,
		},
	}
	rsp, err := p.consumer.GetOneInstance(req)
	if err != nil {
		log.Printf("trpc_report_api_go:polaris getInstances error:%v", err)
		return "", err
	}

	var adds string
	for _, inst := range rsp.Instances {
		adds = net.JoinHostPort(inst.GetHost(), strconv.Itoa(int(inst.GetPort())))
		break
	}

	if adds == "" {
		return "", fmt.Errorf("polaris GetOneInstance fail")
	}

	return adds, nil
}
