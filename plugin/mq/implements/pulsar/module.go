package pulsar

import (
	"github.com/mitchellh/mapstructure"
	"github.com/nearmeng/mango-go/plugin"
	"github.com/nearmeng/mango-go/plugin/mq"
)

var (
	factoryName = "pulsar"
)

type factory struct {
}

func (f *factory) Type() string {
	return "mq"
}

func (f *factory) Name() string {
	return factoryName
}

func (f *factory) Setup(conf map[string]interface{}) (interface{}, error) {
	var config mq.MQConfig
	if err := mapstructure.Decode(conf, &config); err != nil {
		return nil, err
	}

	return NewPulsar(&config)
}

func (f *factory) Destroy(interface{}) error {
	return nil
}

func (f *factory) Reload(i interface{}, conf map[string]interface{}) error {
	var config mq.MQConfig
	if err := mapstructure.Decode(conf, &config); err != nil {
		return err
	}

	kafkaIns := i.(PulsarIns)
	kafkaIns.SetConfig(&config)

	return nil
}

func init() {
	plugin.RegisterPluginFactory(&factory{})
}
