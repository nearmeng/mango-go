package rainbow

import (
	"sync"

	// trpc package
	"git.code.oa.com/trpc-go/trpc-go/config"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
)

var (
	once sync.Once
)

func init() {
	plugin.Register(pluginName, NewPlugin())
}

// NewPlugin 初始化插件
func NewPlugin() plugin.Factory {
	return &rainbowPlugin{}
}

// rainbowPlugin 七彩石插件
type rainbowPlugin struct{}

// Type 返回插件类型
func (p *rainbowPlugin) Type() string {
	return pluginType
}

// PluginConfig trpc-conf插件配置
type PluginConfig struct {
	Providers []*Config `yaml:"providers"`
}

// Setup 加载插件
func (p *rainbowPlugin) Setup(name string, decoder plugin.Decoder) error {

	cfg := &PluginConfig{}
	if err := decoder.Decode(cfg); err != nil {
		return err
	}

	for i, s := range cfg.Providers {
		if err := s.Valid(); err != nil {
			return err
		}
		// 根据不同的 type 创建不同的 stream
		if err := setupProvider(s, i); err != nil {
			return err
		}
	}
	return nil
}

func setupProvider(s *Config, i int) error {
	// 根据不同的 type 创建不同的 stream
	stream, err := NewStream(s)
	if err != nil {
		return err
	}
	// 注册kv
	kv := &KV{
		stream: stream,
		name:   s.Name,
	}
	config.Register(kv)

	if i == 0 {
		config.SetGlobalKV(kv)
	}

	// 注册 provider
	config.RegisterProvider(NewProvider(kv))

	if s.EnableClientProvider {
		once.Do(func() {
			_ = kv.LoadClientConfig()
			go kv.WatchClientConfig()
		})
	}
	return nil
}

// NewStream 新建数据流
func NewStream(cfg *Config) (Stream, error) {
	switch cfg.Type {
	case RainbowTypeTable:
		return NewTableStream(cfg)
	case RainbowTypeGroup:
		return NewGroupStream(cfg)
	default:
		return NewKVStream(cfg)
	}
}

// Response 七彩石的Response实现
type Response struct {
	key   string
	value string
	event config.EventType
	meta  map[string]string
}

// Value 配置项的具体值
func (resp *Response) Value() string {
	return resp.value
}

// MetaData 额外元数据信息
func (resp *Response) MetaData() map[string]string {
	return resp.meta
}

// Event event
func (resp *Response) Event() config.EventType {
	return resp.event
}

// WatchReq watch req
type WatchReq struct {
	key  string
	recv chan Response
	done chan struct{}
}
