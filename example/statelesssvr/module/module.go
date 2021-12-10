package module

import (
	"github.com/nearmeng/mango-go/plugin/log"
	"github.com/nearmeng/mango-go/server_base/app"
)

type TestModule struct {
	testValue int
}

func (m *TestModule) Init() error {

	m.testValue = 1
	log.Info("test module init")

	return nil
}

func (m *TestModule) UnInit() error {
	log.Info("test module uninit")

	return nil
}

func (m *TestModule) Mainloop() {
}

func (m *TestModule) IsPreInit() bool {
	return true
}

func (m *TestModule) GetName() string {
	return "test_module"
}

func (m *TestModule) OnReload() {
	log.Info("test module reload")
}

func init() {
	app.RegisterModule(&TestModule{})
}
