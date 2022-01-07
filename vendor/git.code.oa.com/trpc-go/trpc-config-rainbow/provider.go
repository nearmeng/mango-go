package rainbow

import (
	"context"
	"sync"

	"git.code.oa.com/trpc-go/trpc-go/config"
)

// Provider 七彩石的DataProvider实现
type Provider struct {
	kv    *KV
	recv  chan Response
	cache map[string]bool
	mutex sync.Mutex

	watch chan config.ProviderCallback
}

// NewProvider 初始化
func NewProvider(kv *KV) config.DataProvider {
	p := &Provider{
		kv:    kv,
		recv:  make(chan Response, 1),
		watch: make(chan config.ProviderCallback, 1),
		cache: make(map[string]bool),
	}
	go p.loop()
	return p
}

// Read 读取指定key的配置
func (p *Provider) Read(path string) ([]byte, error) {
	r, err := p.kv.Get(context.TODO(), path)
	if err != nil {
		return nil, err
	}
	go p.watchPath(path)

	return []byte(r.Value()), nil
}

func (p *Provider) watchPath(path string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if ok := p.cache[path]; !ok {
		p.cache[path] = true
		req := &WatchReq{
			recv: make(chan Response),
			done: make(chan struct{}),
			key:  path,
		}
		p.kv.stream.AddWatcher(req)
		go func() {
			for resp := range req.recv {
				p.recv <- resp
			}
		}()
	}
}

// Watch 注册配置变更的回调函数
func (p *Provider) Watch(cb config.ProviderCallback) {
	p.watch <- cb
}

// Name 返回name
func (p *Provider) Name() string {
	return p.kv.Name()
}

func (p *Provider) loop() {
	var fn []config.ProviderCallback
	for {
		select {
		case cb := <-p.watch:
			fn = append(fn, cb)

		case resp := <-p.recv:
			for _, f := range fn {
				go f(resp.key, []byte(resp.Value()))
			}
		}
	}
}
