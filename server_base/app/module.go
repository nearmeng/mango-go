package app

import "fmt"

type ServerModule interface {
	Init() error
	UnInit() error
	Mainloop()

	IsPreInit() bool
	GetName() string

	OnReload()
}

type serverModuleContainer struct {
	moduleCont map[string]ServerModule
}

func (mc *serverModuleContainer) getModuleCount() int {
	return len(mc.moduleCont)
}

func (mc *serverModuleContainer) registerModule(m ServerModule) error {

	mc.moduleCont[m.GetName()] = m
	return nil
}

func (mc *serverModuleContainer) unRegisterModule(name string) error {
	delete(mc.moduleCont, name)
	return nil
}

func (mc *serverModuleContainer) getModule(name string) (ServerModule, error) {

	m, ok := mc.moduleCont[name]
	if !ok {
		return nil, fmt.Errorf("get module %s failed", name)
	}

	return m, nil
}
