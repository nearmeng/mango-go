package rainbow

import (
	"fmt"
	"time"

	"git.code.oa.com/rainbow/golang-sdk/log"
	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/trpc-go/trpc-go"
)

func init() {
	log.SetLevel(log.LOG_LEVEL_NONE)
}

// Config provider 配置
type Config struct {
	EnableClientProvider bool `yaml:"enable_client_provider"`
	OpenSign             bool `yaml:"enable_sign"`

	Addr          string `yaml:"address"`
	Name          string `yaml:"name"`
	AppID         string `yaml:"appid"`
	Group         string `yaml:"group"`
	EnvName       string `yaml:"env_name"`
	Type          string `yaml:"type"`
	Uin           string `yaml:"uin"`
	FileCachePath string `yaml:"file_cache"`
	UserID        string `yaml:"user_id"`
	UserKey       string `yaml:"user_key"`
	HmacWay       string `yaml:"hmac_way" default:"sha256"`
	Timeout       int    `yaml:"timeout" default:"2000"`
}

// Valid 校验配置是否正确
func (cfg *Config) Valid() error {
	return checkInitConf(cfg)
}

// InitOptions 生成初始化配置
func (cfg *Config) InitOptions() []types.AssignInitOption {
	return loadInitOptions(cfg)
}

// GetOptions 生成每次读取的配置
func (cfg *Config) GetOptions() []types.AssignGetOption {
	return loadGetOptions(cfg)
}

// loadInitOptions 生成初始化配置
func loadInitOptions(cfg *Config) []types.AssignInitOption {
	opts := make([]types.AssignInitOption, 0)
	opts = append(
		opts,
		types.IsUsingLocalCache(true),
		types.IsUsingFileCache(true),
		types.AppID(cfg.AppID),
		types.Groups(cfg.Group),
	)
	opts = setAddr(opts, cfg)
	opts = setFileCachePath(opts, cfg)
	opts = setTimeout(opts, cfg.Timeout)
	opts = setEnvName(opts, cfg.EnvName)

	if cfg.OpenSign {
		if cfg.HmacWay != Sha1 && cfg.HmacWay != Sha256 {
			cfg.HmacWay = Sha256
		}
		opts = append(
			opts,
			types.OpenSign(cfg.OpenSign),
			types.UserID(cfg.UserID),
			types.UserKey(cfg.UserKey),
			types.HmacWay(cfg.HmacWay),
		)
	}

	return opts
}

func checkInitConf(cfg *Config) error {
	if cfg.AppID == "" {
		return fmt.Errorf("trpc-config-rainbow: appid not exist")
	}

	if cfg.Group == "" {
		return fmt.Errorf("trpc-config-rainbow: group not exist")
	}

	return checkOpenSignConf(cfg)
}

func checkOpenSignConf(cfg *Config) error {
	if cfg.OpenSign {
		if cfg.UserID == "" {
			return fmt.Errorf("trpc-config-rainbow: user_id not exist")
		}
		if cfg.UserKey == "" {
			return fmt.Errorf("trpc-config-rainbow: user_key not exist")
		}
	}
	return nil
}

func setAddr(opts []types.AssignInitOption, cfg *Config) []types.AssignInitOption {
	if cfg.Addr != "" {
		return append(opts, types.ConnectStr(cfg.Addr))
	}

	return append(opts, types.ConnectStr("http://api.rainbow.oa.com:8080"))
}

func setFileCachePath(opts []types.AssignInitOption, cfg *Config) []types.AssignInitOption {
	if cfg.FileCachePath != "" {
		return append(opts, types.FileCachePath(cfg.FileCachePath))
	}

	return append(opts, types.FileCachePath(fmt.Sprintf("/tmp/%s_%s.dump", cfg.AppID, cfg.Group)))
}

func setTimeout(opts []types.AssignInitOption, t int) []types.AssignInitOption {
	if t > 0 {
		opts = append(opts, types.TimeoutCS(time.Millisecond*time.Duration(t)))
	}

	return opts
}

func setEnvName(opts []types.AssignInitOption, env string) []types.AssignInitOption {
	if env != "" {
		opts = append(opts, types.EnvName(env))
	}

	return opts
}

// loadGetOptions get 配置时的 option
func loadGetOptions(cfg *Config) []types.AssignGetOption {

	opts := make([]types.AssignGetOption, 0)
	opts = append(
		opts,
		types.WithAppID(cfg.AppID),
		types.WithGroup(cfg.Group),
		types.SetNoUpdateCache(false),
	)

	if cfg.Uin != "" {
		opts = append(opts, types.WithUin(cfg.Uin))
	}

	if cfg.EnvName != "" {
		opts = append(opts, types.WithEnvName(cfg.EnvName))
	}

	if trpc.GlobalConfig().Global.EnableSet == "Y" {
		opts = append(opts, types.AddClientInfo("setname", trpc.GlobalConfig().Global.FullSetName))
	}

	return opts
}
