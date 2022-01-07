package tcp

import (
	"github.com/mitchellh/mapstructure"
	"github.com/nearmeng/mango-go/plugin"
	"github.com/spf13/viper"
)

type factory struct {
}

func (f *factory) Type() string {
	return "transport"
}

func (f *factory) Name() string {
	return "tcp"
}

func (f *factory) Setup(v *viper.Viper) (interface{}, error) {
	var config TcpTransportCfg

	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return NewTcpTransport(&config)
}

func (f *factory) Destroy(i interface{}) error {
	return nil
}

func (f *factory) Reload(i interface{}, conf map[string]interface{}) error {
	var config TcpTransportCfg

	if err := mapstructure.Decode(conf, &config); err != nil {
		return err
	}

	tcpTransIns := i.(*TcpTransport)
	tcpTransIns.SetConfig(&config)

	return nil
}

func (f *factory) Mainloop(interface{}) {
}

func init() {
	plugin.RegisterPluginFactory(&factory{})
}
