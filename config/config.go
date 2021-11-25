package config

import (
	"flag"
	"fmt"

	"github.com/spf13/viper"
)

const (
	_defaultConfPath = "./conf/server.toml"
)

type configData struct {
	config      *viper.Viper
	configBak   *viper.Viper
	isUseBak    bool
	cfgFilePath string
	daemonFlag  bool
}

var (
	_config configData
)

func Init() error {
	flag.StringVar(&(_config.cfgFilePath), "conf", _defaultConfPath, "server conf path")
	flag.BoolVar(&(_config.daemonFlag), "daemon", false, "is daemon")
	flag.Parse()

	_config.config = viper.New()
	_config.configBak = viper.New()
	_config.isUseBak = false

	return loadConfig()
}

func getBakConfig() *viper.Viper {
	if _config.isUseBak {
		return _config.config
	}

	return _config.configBak
}

func loadConfig() error {
	v := getBakConfig()

	if _config.cfgFilePath == "" {
		return fmt.Errorf("invalid cfg file path")
	}

	v.SetConfigFile(_config.cfgFilePath)
	v.SetConfigType("toml")

	err := v.ReadInConfig()
	if err != nil {
		return fmt.Errorf("read config failed for %m", err)
	}

	_config.isUseBak = !_config.isUseBak

	fmt.Printf("load config success, filepath %s\n", _config.cfgFilePath)

	return nil
}

func Reload() error {
	return loadConfig()
}

func GetConfig() *viper.Viper {
	if _config.isUseBak {
		return _config.configBak
	}

	return _config.config
}

func GetDaemon() bool {
	return _config.daemonFlag
}