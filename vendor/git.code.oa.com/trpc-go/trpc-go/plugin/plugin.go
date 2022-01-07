// Package plugin 通用插件工厂体系，提供插件注册和装载，主要用于需要通过动态配置加载生成具体插件的情景，不需要配置生成的插件，可直接注册到具体的插件包里面(如codec) 不需要注册到这里。
package plugin

import (
	"errors"
	"fmt"
	"time"

	yaml "gopkg.in/yaml.v3"
)

var (
	// SetupTimeout 每个插件初始化最长超时时间，如果某个插件确实需要加载很久，可以自己修改这里的值
	SetupTimeout = 3 * time.Second

	// MaxPluginSize  最大插件个数
	MaxPluginSize = 1000
)

var (
	plugins = make(map[string]map[string]Factory) // plugin type => { plugin name => plugin factory }
	done    = make(chan struct{})                 // 插件初始化完成通知channel
)

// SetupFinished 发出插件初始化完成通知，个别业务逻辑需要依赖所有插件初始化完成才能继续往下执行。
// 该函数由框架在加载完插件后调用，不是由业务调用。
func SetupFinished() {
	select {
	case <-done: // 已经close过了
	default:
		close(done)
	}
}

// WaitForDone 用户业务逻辑里面可以挂住等待所有插件初始化完成再继续，可自己设置超时时间。
// 当业务逻辑依赖插件时，需要等待所有插件初始化完成才能执行，可调用该函数。
func WaitForDone(timeout time.Duration) bool {
	select {
	case <-done:
		return true
	case <-time.After(timeout):
	}
	return false
}

// Factory 插件工厂统一抽象，外部插件需要实现该接口，通过该工厂接口生成具体的插件并注册到具体的插件类型里面。
type Factory interface {
	// Type 插件的类型 如 selector log config tracing
	Type() string
	// Setup 根据配置项节点装载插件，需要用户自己先定义好具体插件的配置数据结构
	Setup(name string, dec Decoder) error
}

// Decoder 节点配置解析器。
type Decoder interface {
	Decode(cfg interface{}) error // 输入参数为自定义的配置数据结构
}

// YamlNodeDecoder yaml节点配置解析器。
type YamlNodeDecoder struct {
	Node *yaml.Node
}

// Decode 解析yaml node配置。
func (d *YamlNodeDecoder) Decode(cfg interface{}) error {
	if d.Node == nil {
		return errors.New("yaml node empty")
	}
	return d.Node.Decode(cfg)
}

// Register 注册插件工厂 可自己指定插件名，支持相同的实现，不同的配置注册不同的工厂实例。
func Register(name string, f Factory) {
	factories, ok := plugins[f.Type()]
	if !ok {
		factories = make(map[string]Factory)
		plugins[f.Type()] = factories
	}
	factories[name] = f
}

// Get 根据插件类型，插件名字获取插件工厂。
func Get(typ string, name string) Factory {
	return plugins[typ][name]
}

// Config 插件统一配置 plugin type => { plugin name => plugin config } 。
type Config map[string]map[string]yaml.Node

// Setup 通过配置生成并装载具体插件。
func (c Config) Setup() error {
	// 从框架配置文件中逐个取出插件并放进有序的插件队列中
	sortedPlugins, setupStatus, err := c.loadPluginInfos()
	if err != nil {
		return err
	}

	// 从插件队列中按顺序逐个取出插件并初始化
	if err := c.setupPlugins(sortedPlugins, setupStatus); err != nil {
		return err
	}
	return nil
}

func (c Config) loadPluginInfos() (chan pluginInfo, map[string]bool, error) {
	var (
		sortedPlugins = make(chan pluginInfo, MaxPluginSize) // 使用channel初始化插件队列，方便后面按顺序逐个加载插件
		setupStatus   = make(map[string]bool)                // 插件初始化状态，plugin key => true初始化完成 false未初始化
	)
	for typ, factories := range c {
		for name, cfg := range factories {
			factory := Get(typ, name)
			if factory == nil {
				return nil, nil, fmt.Errorf("plugin %s:%s no registered or imported, do not configure", typ, name)
			}
			p := pluginInfo{
				factory: factory,
				typ:     typ,
				name:    name,
				cfg:     cfg,
			}
			select {
			case sortedPlugins <- p:
			default:
				return nil, nil, fmt.Errorf("plugin number exceed max limit:%d", len(sortedPlugins))
			}
			setupStatus[p.key()] = false
		}
	}
	return sortedPlugins, setupStatus, nil
}

func (c Config) setupPlugins(sortedPlugins chan pluginInfo, setupStatus map[string]bool) error {
	num := len(sortedPlugins)
	for num > 0 {
		for i := 0; i < num; i++ {
			p := <-sortedPlugins
			// 先判断当前插件依赖的其他插件是否已经初始化完成
			if deps, err := p.hasDependence(setupStatus); err != nil {
				return err
			} else if deps { // 被依赖的插件还未初始化，将当前插件移到channel末尾
				sortedPlugins <- p
				continue
			}
			if err := p.setup(); err != nil {
				return err
			}
			setupStatus[p.key()] = true
		}
		if len(sortedPlugins) == num { // 取出来又原封不动塞回去，说明没有一个插件setup成功，循环依赖了，导致无插件可以初始化，返回失败
			return fmt.Errorf("cycle depends, not plugin is setup")
		}
		num = len(sortedPlugins) // 继续处理插入到channel末尾的插件
	}
	return nil
}

// Depender 依赖接口，由具体实现插件决定是否有依赖其他插件, 需要保证被依赖的插件先初始化完成。
type Depender interface {
	// DependsOn 假如一个插件依赖另一个插件，则返回被依赖的插件的列表：数组元素为 type-name 如 [ "selector-polaris" ]
	DependsOn() []string
}

// FlexDepender 弱依赖接口，如果被依赖的插件存在，才去保证被依赖的插件先初始化完成
type FlexDepender interface {
	FlexDependsOn() []string
}

// pluginInfo 插件信息。
type pluginInfo struct {
	factory Factory
	typ     string
	name    string
	cfg     yaml.Node
}

// 判断是否有依赖的插件未初始化过。
// 输入参数为所有插件的初始化状态。
// 输出参数bool true被依赖的插件未初始化完成，仍有依赖，false没有依赖其他插件或者被依赖的插件已经初始化完成
func (p *pluginInfo) hasDependence(setupStatus map[string]bool) (bool, error) {
	deps, ok := p.factory.(Depender)
	if ok {
		hasDeps, err := p.checkDependence(setupStatus, deps.DependsOn(), false)
		if err != nil {
			return false, err
		}
		if hasDeps { // 个别插件会同时强依赖和弱依赖多个不同插件，当所有强依赖满足后需要再判断弱依赖关系
			return true, nil
		}
	}
	fd, ok := p.factory.(FlexDepender)
	if ok {
		return p.checkDependence(setupStatus, fd.FlexDependsOn(), true)
	}
	// 该插件不依赖任何其他插件
	return false, nil
}

func (p *pluginInfo) checkDependence(setupStatus map[string]bool, dependences []string, flexible bool) (bool, error) {
	for _, name := range dependences {
		if name == p.key() {
			return false, errors.New("plugin not allowed to depend on itself")
		}
		setup, ok := setupStatus[name]
		if !ok {
			if flexible {
				continue
			}
			return false, fmt.Errorf("depends plugin %s not exists", name)
		}
		if !setup {
			return true, nil
		}
	}
	return false, nil
}

// setup 初始化单个插件。
func (p *pluginInfo) setup() error {
	var (
		ch  = make(chan struct{})
		err error
	)
	go func() {
		err = p.factory.Setup(p.name, &YamlNodeDecoder{Node: &p.cfg})
		close(ch)
	}()
	select {
	case <-ch:
	case <-time.After(SetupTimeout):
		return fmt.Errorf("setup plugin %s timeout", p.key())
	}
	if err != nil {
		return fmt.Errorf("setup plugin %s error: %v", p.key(), err)
	}
	return nil
}

// key 插件的唯一索引：type-name 。
func (p *pluginInfo) key() string {
	return fmt.Sprintf("%s-%s", p.typ, p.name)
}
