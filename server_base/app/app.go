package app

import (
	"os"
	"syscall"
	"time"

	"github.com/nearmeng/mango-go/common/process"
	"github.com/nearmeng/mango-go/common/signal"
	"github.com/nearmeng/mango-go/config"
	"github.com/nearmeng/mango-go/plugin"
	"github.com/nearmeng/mango-go/plugin/log"

	_ "github.com/nearmeng/mango-go/plugin/log/bingologger"
	_ "github.com/nearmeng/mango-go/plugin/mq/kafka"
	_ "github.com/nearmeng/mango-go/plugin/mq/pulsar"
	_ "github.com/nearmeng/mango-go/plugin/transport/tcp"
)

var (
	_serverApp  *serverApp            = nil
	_moduleCont serverModuleContainer = serverModuleContainer{
		moduleCont: make(map[string]ServerModule),
	}

	_serverFrame   = 4
	_finishChannel = make(chan struct{})
)

type serverApp struct {
	serverName     string
	serverID       string
	lastReloadTime int64
}

func NewServerApp(name string) *serverApp {
	s := serverApp{
		serverName:     name,
		serverID:       "",
		lastReloadTime: 0,
	}

	_serverApp = &s
	return _serverApp
}

func GetServerApp() *serverApp {
	return _serverApp
}

func (s *serverApp) GetServerName() string {
	return s.serverName
}

func (s *serverApp) GetServerID() string {
	return s.serverID
}

func (s *serverApp) initInputParam() error {
	switch config.GetCommand() {
	case "stop":
		process.SendSignal(s.serverID, syscall.SIGUSR1)
		os.Exit(0)
	case "reload":
		process.SendSignal(s.serverID, syscall.SIGUSR2)
		os.Exit(0)
	case "start":
		break
	}

	return nil
}

func (s *serverApp) Init() error {
	//config
	err := config.Init()
	if err != nil {
		return err
	}

	conf := config.GetConfig()
	s.serverID = conf.GetString("svrinfo.serverid")

	//input param process
	err = s.initInputParam()
	if err != nil {
		return err
	}

	//kill pre process
	process.KillPre(s.serverID)

	//signal
	signal.RegisterSignalHandler([]os.Signal{syscall.SIGUSR1}, s.Quit)
	signal.RegisterSignalHandler([]os.Signal{syscall.SIGUSR2}, s.Reload)
	signal.StartSignal()

	//plugin
	err = plugin.InitPlugin(conf.Sub("plugin"))
	if err != nil {
		return err
	}

	//module
	for _, module := range _moduleCont.moduleCont {
		if module.IsPreInit() {
			err = module.Init()
			if err != nil {
				return err
			}
		}
	}

	for _, module := range _moduleCont.moduleCont {
		if !module.IsPreInit() {
			err = module.Init()
			if err != nil {
				return err
			}
		}
	}

	log.Info("server %s %s init success", s.serverName, s.serverID)

	return nil
}

func (s *serverApp) Fini() error {

	//module
	for _, module := range _moduleCont.moduleCont {
		if module.IsPreInit() {
			err := module.UnInit()
			if err != nil {
				return err
			}
		}
	}

	for _, module := range _moduleCont.moduleCont {
		if !module.IsPreInit() {
			err := module.UnInit()
			if err != nil {
				return err
			}
		}
	}

	log.Info("server %s fini success", s.serverName)
	return nil
}

func (s *serverApp) Mainloop() {

	interval := time.Second / time.Duration(_serverFrame)
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		finished := false

		select {
		case <-_finishChannel:
			log.Info("server %s finished", s.serverName)
			finished = true
		case curr := <-t.C:
			s.onFrame(curr)
		}

		if finished {
			break
		}
	}
}

func (s *serverApp) onFrame(t time.Time) {

	for _, module := range _moduleCont.moduleCont {
		module.Mainloop()
	}

}

func (s *serverApp) Quit() {
	log.Info("recv signal to quit")

	_finishChannel <- struct{}{}
}

func (s *serverApp) Reload() {

	log.Info("server %s reload begin", s.serverName)

	err := config.Reload()
	if err != nil {
		log.Error("config reload failed for %m", err)
	}

	conf := config.GetConfig()
	err = plugin.ReloadPlugin(conf)
	if err != nil {
		log.Error("plugin reload failed for %m", err)
	}

	for _, module := range _moduleCont.moduleCont {
		module.OnReload()
	}

	log.Info("server %s reload end", s.serverName)
}

func RegisterModule(m ServerModule) error {
	return _moduleCont.registerModule(m)
}

func GetModule(name string) (ServerModule, error) {
	return _moduleCont.getModule(name)
}

func GetModuleCount() int {
	return _moduleCont.getModuleCount()
}
