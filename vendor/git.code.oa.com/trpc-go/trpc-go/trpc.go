// Package trpc 是公司统一微服务框架的 golang 版本，主要是以高性能，可插拔，易测试为出发点而设计的 rpc 框架。
package trpc

import (
	"fmt"

	"git.code.oa.com/trpc-go/trpc-go/admin"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/naming/registry"
	"git.code.oa.com/trpc-go/trpc-go/server"

	"go.uber.org/automaxprocs/maxprocs"
)

// NewServer 读取 yaml 配置文件快速启动支持多 service 的 server。
// 默认的配置文件为./trpc_go.yaml，您也可以通过`-conf`选项指定配置文件。
// 该函数内部会自动调用 flag.Parse 解析命令行参数，不需要用户再显示解析flag。
// 该函数全局只允许调用一次。
func NewServer(opt ...server.Option) *server.Server {
	// 获取服务启动配置文件路径
	path := serverConfigPath()

	// 解析框架配置
	cfg, err := LoadConfig(path)
	if err != nil {
		panic("load config fail: " + err.Error())
	}

	// 保存到全局配置里面，方便其他插件获取配置数据
	SetGlobalConfig(cfg)

	// 加载插件
	if err := Setup(cfg); err != nil {
		panic("setup plugin fail: " + err.Error())
	}

	// 默认配置 GOMAXPROCS，避免在容器场景下出现问题
	maxprocs.Set(maxprocs.Logger(log.Debugf))
	return NewServerWithConfig(cfg, opt...)
}

// NewServerWithConfig 使用自定义配置初始化一个服务。
// 如用户不用默认的 yaml 配置文件，可以自己解析好自己的配置文件后构造出 Config 结构体并调用该函数启动服务。
// 通过自定义配置文件启动服务时，插件初始化逻辑需要自己实现。
func NewServerWithConfig(cfg *Config, opt ...server.Option) *server.Server {
	// 修复配置数据，填充默认值
	if err := RepairConfig(cfg); err != nil {
		panic("repair config fail: " + err.Error())
	}

	// 保存到全局配置里面，再设置一次，防止用户忘记设置
	SetGlobalConfig(cfg)

	s := &server.Server{}

	// 初始化admin设置
	setupAdmin(s, cfg)

	// 逐个 service 加载配置
	for _, c := range cfg.Server.Service {
		s.AddService(c.Name, newServiceWithConfig(cfg, c, opt...))
	}
	return s
}

func setupAdmin(s *server.Server, cfg *Config) {
	// 有配置 admin 参数则启动 admin 服务
	if cfg.Server.Admin.Port == 0 {
		return
	}

	opts := []admin.Option{
		admin.WithVersion(Version()),
		admin.WithAddr(fmt.Sprintf("%s:%d", cfg.Server.Admin.IP, cfg.Server.Admin.Port)),
		admin.WithTLS(cfg.Server.Admin.EnableTLS),
		admin.WithConfigPath(ServerConfigPath),
		admin.WithReadTimeout(getMillisecond(cfg.Server.Admin.ReadTimeout)),
		admin.WithWriteTimeout(getMillisecond(cfg.Server.Admin.WriteTimeout)),
	}

	s.AddService("admin", admin.NewTrpcAdminServer(opts...))
}

func newServiceWithConfig(cfg *Config, serviceCfg *ServiceConfig, opt ...server.Option) server.Service {
	var filters []filter.Filter
	for _, name := range Deduplicate(cfg.Server.Filter, serviceCfg.Filter) { // 全局 filter 在前，且去重
		f := filter.GetServer(name)
		if f == nil {
			panic(fmt.Sprintf("filter %s no registered, do not configure", name))
		}
		filters = append(filters, f)
	}

	// 检查 service 对应的 registry
	reg := registry.Get(serviceCfg.Name)
	if serviceCfg.Registry != "" && reg == nil {
		log.Warnf("service:%s registry not exist", serviceCfg.Name)
	}

	opts := []server.Option{
		server.WithNamespace(cfg.Global.Namespace),
		server.WithEnvName(cfg.Global.EnvName),
		server.WithServiceName(serviceCfg.Name),
		server.WithProtocol(serviceCfg.Protocol),
		server.WithNetwork(serviceCfg.Network),
		server.WithAddress(serviceCfg.Address),
		server.WithFilters(filters),
		server.WithRegistry(reg),
		server.WithDisableRequestTimeout(serviceCfg.DisableRequestTimeout),
		server.WithTimeout(getMillisecond(serviceCfg.Timeout)),
		server.WithCloseWaitTime(getMillisecond(cfg.Server.CloseWaitTime)),
		server.WithIdleTimeout(getMillisecond(serviceCfg.Idletime)),
		server.WithTLS(serviceCfg.TLSCert, serviceCfg.TLSKey, serviceCfg.CACert),
		server.WithServerAsync(*serviceCfg.ServerAsync),
		server.WithMaxRoutines(serviceCfg.MaxRoutines),
		server.WithWritev(*serviceCfg.Writev),
		server.WithOverloadCtrl(&serviceCfg.OverloadCtrl),
	}
	if cfg.Global.EnableSet == "Y" {
		opts = append(opts, server.WithSetName(cfg.Global.FullSetName))
	}
	opts = append(opts, opt...)
	return server.New(opts...)
}
