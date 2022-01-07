package trpc

import (
	"errors"

	"github.com/nearmeng/mango-go/plugin"
	"github.com/spf13/viper"
)

type factory struct {
}

func (f *factory) Type() string {
	return "rpc"
}

func (f *factory) Name() string {
	return "trpc"
}

func (f *factory) Setup(v *viper.Viper) (interface{}, error) {
	var config TrpcConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	ts := NewTrpcServer(&config)
	err := ts.Init()

	return ts, err
}

func (f *factory) Destroy(i interface{}) error {
	ts, ok := i.(*TrpcServer)
	if ts == nil || !ok {
		return errors.New("invalid trpcserver")
	}

	err := ts.UnInit()
	if err != nil {
		return errors.New("trpc server uninit failed")
	}
	return nil
}

func (f *factory) Reload(i interface{}, conf map[string]interface{}) error {
	return nil
}

func (f *factory) Mainloop(i interface{}) {
	ts, ok := i.(*TrpcServer)
	if ts != nil && ok {
		go ts.GetServer().Serve()
	}
}

func init() {
	plugin.RegisterPluginFactory(&factory{})
}
