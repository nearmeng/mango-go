package plugin

import (
	"fmt"
	"sync"

	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/spf13/viper"
)

type PluginConfig map[string]map[string]interface{}

type PluginFactory interface {
	Type() string
	Name() string
	Setup(*viper.Viper) (interface{}, error)
	Destroy(interface{}) error
	Reload(interface{}, map[string]interface{}) error
	Mainloop(interface{})
}

var (
	_pluginFactoryMgr  = make(map[string]PluginFactory)
	_pluginFactoryLock = sync.RWMutex{}
	_pluginMgr         = make(map[string]interface{})
)

func RegisterPluginFactory(f PluginFactory) {
	_pluginFactoryLock.Lock()
	defer _pluginFactoryLock.Unlock()

	key := constructPluginKey(f.Type(), f.Name())
	_pluginFactoryMgr[key] = f

	log.Info("register plugin factory, key %s", key)
}

func GetPluginInst(typ string, name string) interface{} {
	key := constructPluginKey(typ, name)

	plugin, ok := _pluginMgr[key]
	if !ok {
		return nil
	}

	return plugin
}

func registerPluginInst(key string, plugin interface{}) {
	_pluginMgr[key] = plugin

	log.Info("register plugin inst, key %s", key)
}

func constructPluginKey(typ string, name string) string {
	return fmt.Sprintf("%s_%s", typ, name)
}

func Init(v *viper.Viper) error {
	var cfg PluginConfig

	err := v.Unmarshal(&cfg)
	if err != nil {
		return fmt.Errorf("unmarshal failed for %w", err)
	}

	//init log first
	s, ok := cfg["log"]
	if ok {
		for n, _ := range s {
			f := getPluginFactory("log", n)
			if f != nil {
				plugin, err := f.Setup(v.Sub("log").Sub(n))
				if err != nil {
					return fmt.Errorf("log plugin init failed")
				}

				registerPluginInst(constructPluginKey("log", n), plugin)
			}
		}
	}

	for t, s := range cfg {
		if t == "log" {
			continue
		}

		for n, _ := range s {
			f := getPluginFactory(t, n)
			if f == nil {
				return fmt.Errorf("get plugin factory failed, type %s name %s", t, n)
			}

			log.Info("init plugin %s %s", t, n)

			plugin, err := f.Setup(v.Sub(t).Sub(n))
			if err != nil {
				return fmt.Errorf("plugin setup failed, type %s name %s", t, n)
			}

			registerPluginInst(constructPluginKey(t, n), plugin)
		}
	}

	return nil
}

func Reload(v *viper.Viper) error {
	return nil
}

func Mainloop() {
	for k, p := range _pluginMgr {
		f := getPluginFactoryByKey(k)
		go f.Mainloop(p)
	}
}

func Destroy() error {
	for k, p := range _pluginMgr {
		log.Info("begin destroy plugin %s", k)

		f := getPluginFactoryByKey(k)
		f.Destroy(p)
	}

	return nil
}

func getPluginFactory(t string, n string) PluginFactory {
	_pluginFactoryLock.RLock()
	defer _pluginFactoryLock.RUnlock()

	key := constructPluginKey(t, n)
	return _pluginFactoryMgr[key]
}

func getPluginFactoryByKey(key string) PluginFactory {
	_pluginFactoryLock.RLock()
	defer _pluginFactoryLock.RUnlock()

	return _pluginFactoryMgr[key]
}
