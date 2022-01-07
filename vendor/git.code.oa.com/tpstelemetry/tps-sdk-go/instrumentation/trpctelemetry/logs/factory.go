// Copyright 2020 The TpsTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logs 日志组件
package logs

import (
	"errors"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
)

const pluginType = "log"

func init() {
	plugin.Register("default", &Factory{})
}

// Factory 日志插件工厂 由框架启动读取配置文件 调用该工厂生成具体日志
type Factory struct {
}

// Type 日志插件类型
func (f *Factory) Type() string {
	return pluginType
}

// Setup 启动加载log配置 并注册日志
func (f *Factory) Setup(name string, configDec plugin.Decoder) error {
	if configDec == nil {
		return errors.New("log config decoder empty")
	}

	conf, callerSkip, err := f.setupConfig(configDec)
	if err != nil {
		return err
	}

	hasTpstelemetry := false
	for _, c := range conf {
		if c.Writer == "tpstelemetry" {
			hasTpstelemetry = true
			break
		}
	}

	if !hasTpstelemetry {
		conf = append(conf, log.OutputConfig{
			Writer: "tpstelemetry",
		})
	}

	logger := log.NewZapLogWithCallerSkip(conf, callerSkip)
	if logger == nil {
		return errors.New("new zap logger fail")
	}

	log.Register(name, logger)

	if name == "default" {
		log.SetLogger(logger)
	}

	return nil
}

func (f *Factory) setupConfig(configDec plugin.Decoder) (log.Config, int, error) {
	conf := log.Config{}

	err := configDec.Decode(&conf)
	if err != nil {
		return nil, 0, err
	}

	if len(conf) == 0 {
		return nil, 0, errors.New("log config output empty")
	}

	// 如果没有配置caller skip，则默认为2
	callerSkip := 2
	for i := 0; i < len(conf); i++ {
		if conf[i].CallerSkip != 0 {
			callerSkip = conf[i].CallerSkip
		}
	}
	return conf, callerSkip, nil
}
