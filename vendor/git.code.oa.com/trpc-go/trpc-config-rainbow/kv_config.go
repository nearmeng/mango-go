package rainbow

import (
	"context"
	"fmt"

	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/trpc-go/trpc-go/config"

	"gopkg.in/yaml.v3"
)

// KV 七彩石的KvConfig实现
type KV struct {
	stream Stream
	name   string
}

// Name 返回name
func (k *KV) Name() string {
	return k.name
}

// Get 拉取配置
func (k *KV) Get(ctx context.Context, key string, opts ...config.Option) (config.Response, error) {
	if err := k.stream.Check(); err != nil {
		return nil, err
	}
	resp, err := k.stream.Get(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %s, %s", ErrConfigNotExist, key, err.Error())
	}
	return resp, nil
}

// Put 更新配置操作
func (k *KV) Put(ctx context.Context, key string, val string, opts ...config.Option) error {
	return ErrUnsupportedOperation
}

// Del 删除配置操作
func (k *KV) Del(ctx context.Context, key string, opts ...config.Option) error {
	return ErrUnsupportedOperation
}

// Watch 监听配置变更
func (k *KV) Watch(ctx context.Context, key string, opts ...config.Option) (<-chan config.Response, error) {
	if err := k.stream.Check(); err != nil {
		return nil, err
	}
	c := make(chan config.Response, 1)

	req := &WatchReq{
		key:  key,
		recv: make(chan Response, 1),
		done: make(chan struct{}),
	}

	if res, err := k.Get(ctx, key, opts...); err == nil {
		resp := Response{
			key:   key,
			value: res.Value(),
			event: config.EventTypePut,
		}
		req.recv <- resp
	}

	k.stream.AddWatcher(req)
	go k.watch(key, req.recv, c)

	<-req.done
	return c, nil
}

func (k *KV) watch(key string, in chan Response, out chan config.Response) {
	defer close(out)
	var tmp Response

	for resp := range in {
		if !isChange(resp, tmp) {
			continue
		}

		if resp.key == key || k.dataTypeNotKV() {
			out <- &Response{
				key:   resp.key,
				event: resp.event,
				value: resp.value,
			}

			tmp.key = resp.key
			tmp.event = resp.event
			tmp.value = resp.value
		}
	}

}

func isChange(r1, r2 Response) bool {
	if r1.key == r2.key && r1.event == r2.event && r1.value == r2.value {
		return false
	}

	return true
}

func (k *KV) dataTypeNotKV() bool {
	return k.stream.DataType() == RainbowTypeTable || k.stream.DataType() == RainbowTypeGroup
}

// LoadClientConfig 加载 rainbow 中client.yaml 配置变更并将配置注册到client中
func (k *KV) LoadClientConfig() error {
	resp, err := k.Get(context.TODO(), "client.yaml")
	if err != nil {
		return err
	}

	if resp.Event() == config.EventTypePut {
		return applyClientConfig([]byte(resp.Value()))
	}

	return nil
}

// WatchClientConfig 监听 rainbow 中 client.yaml 配置变更并将配置注册到client中
func (k *KV) WatchClientConfig() {

	// NOTE: 将默认配置复制一份，用户清空远程配置时，回退使用默认配置.
	defClientConfig := make(map[string]*client.BackendConfig)
	for k, s := range client.DefaultClientConfig() {
		defClientConfig[k] = s
	}

	c, _ := k.Watch(context.TODO(), "client.yaml")

	for resp := range c {

		// NOTE: 如果tconf中配置被清空或删除，使用框架配置
		if resp.Event() == config.EventTypeDel || len(resp.Value()) == 0 {
			client.RegisterConfig(defClientConfig)
			continue
		}

		_ = applyClientConfig([]byte(resp.Value()))
	}

}

func applyClientConfig(buf []byte) error {

	cfg := &trpc.Config{}
	cfg.Client.Network = "tcp"
	cfg.Client.Protocol = "trpc"
	if err := yaml.Unmarshal(buf, cfg); err != nil {
		return err
	}

	trpc.RepairConfig(cfg)
	serviceConfigMap := make(map[string]*client.BackendConfig)
	serviceConfigMap["*"] = &client.BackendConfig{
		Network:        cfg.Client.Network,
		Protocol:       cfg.Client.Protocol,
		Namespace:      cfg.Client.Namespace,
		Timeout:        cfg.Client.Timeout,
		Filter:         cfg.Client.Filter,
		Discovery:      cfg.Client.Discovery,
		Loadbalance:    cfg.Client.Loadbalance,
		Circuitbreaker: cfg.Client.Circuitbreaker,
	}

	for _, s := range cfg.Client.Service {
		serviceConfigMap[s.Callee] = s
	}

	client.RegisterConfig(serviceConfigMap)
	return nil
}
