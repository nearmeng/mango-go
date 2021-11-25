package plugin

import (
	"fmt"
	"sync"

	"github.com/spf13/viper"
)

type PluginConfig struct {
	Plugin map[string]map[string]map[string]interface{} `toml:"plugin"`
}

type PluginFactory interface {
	Type() string
	Name() string
	Setup(map[string]interface{}) (interface{}, error)
	Destroy(interface{}) error
	Reload(interface{}, map[string]interface{}) error
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

	fmt.Printf("register plugin factory, key %s\n", key)
}

func GetPluginInst(typ string, name string) interface{} {
	key := constructPluginKey(typ, name)

	plugin, ok := _pluginMgr[key]
	if !ok {
		return nil
	}

	return &plugin
}

func registerPluginInst(key string, plugin interface{}) {
	_pluginMgr[key] = plugin
}

func constructPluginKey(typ string, name string) string {
	return fmt.Sprintf("%s_%s", typ, name)
}

func InitPlugin(v *viper.Viper) error {
	var cfg PluginConfig

	err := v.Unmarshal(&cfg)
	if err != nil {
		return fmt.Errorf("unmarshal failed for %w", err)
	}

	for t, s := range cfg.Plugin {
		for n, c := range s {
			f := getPluginFactory(t, n)
			if f == nil {
				return fmt.Errorf("get plugin factory failed, type %s name %s", t, n)
			}

			fmt.Printf("init plugin %s %s\n", t, n)

			plugin, error := f.Setup(c)
			if error != nil {
				return fmt.Errorf("plugin setup failed, type %s name %s", t, n)
			}

			registerPluginInst(constructPluginKey(t, n), plugin)
		}
	}

	return nil
}

func ReloadPlugin(v *viper.Viper) error {
	return nil
}

func getPluginFactory(t string, n string) PluginFactory {
	_pluginFactoryLock.RLock()
	defer _pluginFactoryLock.RUnlock()

	key := constructPluginKey(t, n)
	return _pluginFactoryMgr[key]
}
