package registry

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"git.code.oa.com/polaris/polaris-go/api"
	plog "git.code.oa.com/polaris/polaris-go/pkg/log"
	"git.code.oa.com/polaris/polaris-go/pkg/model"
	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
)

const (
	defaultHeartBeat = 5
	defaultWeight    = 100
	defaultTTL       = 5
)

// Registry 服务注册
type Registry struct {
	provider api.ProviderAPI
	cfg      *Config
	host     string
	port     int
}

// newRegistry 新建实例
func newRegistry(provider api.ProviderAPI, cfg *Config) (*Registry, error) {
	if len(cfg.ServiceToken) == 0 {
		return nil, fmt.Errorf("service: %s, token can not be empty", cfg.ServiceName)
	}
	if cfg.HeartBeat == 0 {
		cfg.HeartBeat = defaultHeartBeat
	}
	if cfg.Weight == 0 {
		cfg.Weight = defaultWeight
	}
	if cfg.TTL == 0 {
		cfg.TTL = defaultTTL
	}
	return &Registry{
		provider: provider,
		cfg:      cfg,
	}, nil
}

// NewRegistry 提供外部可访问的新建实例接口
func NewRegistry(provider api.ProviderAPI, cfg *Config) (*Registry, error) {
	return newRegistry(provider, cfg)
}

// Register 注册服务
func (r *Registry) Register(_ string, opt ...registry.Option) error {
	opts := &registry.Options{}
	for _, o := range opt {
		o(opts)
	}
	address := opts.Address
	if r.cfg.BindAddress != "" {
		address = parseAddress(r.cfg.BindAddress)
	}
	host, portRaw, _ := net.SplitHostPort(address)
	port, _ := strconv.ParseInt(portRaw, 10, 64)
	r.host = host
	r.port = int(port)
	if r.cfg.EnableRegister {
		if err := r.register(); err != nil {
			return err
		}
	}
	r.heartBeat()
	return nil
}

func (r *Registry) register() error {
	req := &api.InstanceRegisterRequest{
		InstanceRegisterRequest: model.InstanceRegisterRequest{
			Namespace:    r.cfg.Namespace,
			Service:      r.cfg.ServiceName,
			Host:         r.host,
			Port:         r.port,
			ServiceToken: r.cfg.ServiceToken,
			Weight:       &r.cfg.Weight,
			Metadata:     r.cfg.Metadata,
		},
	}
	if !r.cfg.DisableHealthCheck {
		req.SetTTL(r.cfg.TTL)
	}
	resp, err := r.provider.Register(req)
	if err != nil {
		return fmt.Errorf("fail to Register instance, err is %v", err)
	}
	plog.GetBaseLogger().Debugf("success to register instance1, id is %s\n", resp.InstanceID)
	r.cfg.InstanceID = resp.InstanceID
	return nil
}

func (r *Registry) heartBeat() {
	tick := time.Second * time.Duration(r.cfg.HeartBeat)
	go func() {
		for {
			req := &api.InstanceHeartbeatRequest{
				InstanceHeartbeatRequest: model.InstanceHeartbeatRequest{
					Service:      r.cfg.ServiceName,
					ServiceToken: r.cfg.ServiceToken,
					Namespace:    r.cfg.Namespace,
					InstanceID:   r.cfg.InstanceID,
					Host:         r.host,
					Port:         r.port,
				},
			}
			if err := r.provider.Heartbeat(req); err != nil {
				plog.GetBaseLogger().Errorf("heartbeat report err: %v\n", err)
			} else {
				plog.GetBaseLogger().Debugf("heart beat success")
			}
			time.Sleep(tick)
		}
	}()
}

// Deregister 反注册
func (r *Registry) Deregister(_ string) error {
	if !r.cfg.EnableRegister {
		return nil
	}
	req := &api.InstanceDeRegisterRequest{
		InstanceDeRegisterRequest: model.InstanceDeRegisterRequest{
			Service:      r.cfg.ServiceName,
			Namespace:    r.cfg.Namespace,
			InstanceID:   r.cfg.InstanceID,
			ServiceToken: r.cfg.ServiceToken,
			Host:         r.host,
			Port:         r.port,
		},
	}
	if err := r.provider.Deregister(req); err != nil {
		return fmt.Errorf("deregister error: %s", err.Error())
	}
	return nil
}

// parseAddress 解析地址
func parseAddress(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return address
	}

	// host 不是 ip
	parsedIP := trpc.GetIP(host)
	return net.JoinHostPort(parsedIP, port)
}
