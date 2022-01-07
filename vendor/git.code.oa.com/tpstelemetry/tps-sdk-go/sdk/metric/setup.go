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

// Package metric metric 子系统
package metric

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"git.code.oa.com/tpstelemetry/tpstelemetry-protocol/tpstelemetry/proto/operation"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/remote"
)

// Setup setup metrics
func Setup(opts ...SetupOption) error {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}

	return SetupByConfig(*cfg)
}

// SetupByConfig setup by config
func SetupByConfig(cfg Config) error {
	if !cfg.Enabled {
		return nil
	}

	if len(cfg.RegistryEndpoints) == 0 {
		return errors.New("metric: registry endpoints nil")
	}
	if cfg.Instance.TenantID == "" {
		return errors.New("metric: tenant id nil")
	}
	if cfg.Instance.Addr == "" {
		return errors.New("metric: exporter addr nil")
	}
	if cfg.TTL == 0 {
		cfg.TTL = DefaultRegisterTTL
	}
	if cfg.ServerOwner != "" {
		serverMetadata.WithLabelValues(cfg.ServerOwner, cfg.CmdbID).Set(1)
	}
	atomicCodeMapping := &atomic.Value{}
	atomicCodeMapping.Store(covertCodesToMapping(cfg.Codes))
	DefaultCodeTypeFunc = genCodeTypeFunc(atomicCodeMapping)
	if cfg.Configurator != nil {
		cfg.Configurator.RegisterConfigApplyFunc(genConfigApplyFunc(cfg, atomicCodeMapping))
	}
	reg := NewEtcdRegistry(cfg.RegistryEndpoints)
	_, err := reg.Register(context.Background(), &cfg.Instance, cfg.TTL)
	return err
}

// SetupOption Setup func
type SetupOption func(config *Config)

// WithEnabled 设置开启
func WithEnabled(enabled bool) SetupOption {
	return func(config *Config) {
		config.Enabled = enabled
	}
}

// WithInstance 指定配置实例
func WithInstance(ins *Instance) SetupOption {
	return func(config *Config) {
		config.Instance = *ins
	}
}

// WithTTL 指定 ttl
func WithTTL(ttl time.Duration) SetupOption {
	return func(config *Config) {
		config.TTL = ttl
	}
}

// WithRegistryEndpoints 指定注册endpoint
func WithRegistryEndpoints(endpoints []string) SetupOption {
	return func(config *Config) {
		config.RegistryEndpoints = endpoints
	}
}

// WithServerOwner ...
func WithServerOwner(serverOwner string) SetupOption {
	return func(config *Config) {
		config.ServerOwner = serverOwner
	}
}

// WithCmdbID ...
func WithCmdbID(cmdbID string) SetupOption {
	return func(config *Config) {
		config.CmdbID = cmdbID
	}
}

// WithCodes 错误码特例
func WithCodes(codes []*Code) SetupOption {
	return func(config *Config) {
		config.Codes = codes
	}
}

// WithConfigurator 配置器
func WithConfigurator(configurator remote.Configurator) SetupOption {
	return func(config *Config) {
		config.Configurator = configurator
	}
}

var (
	// serverMetadata 服务元信息metric
	serverMetadata = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "tpstelemetry_sdk",
		Name:      "server_metadata",
		Help:      "服务元信息",
	}, []string{"server_owner", "cmdb_id"})
)

func covertCodesToMapping(codes []*Code) map[string][]*Code {
	codeMapping := make(map[string][]*Code, len(codes))
	for _, v := range codes {
		codeMapping[v.Code] = append(codeMapping[v.Code], v)
	}
	return codeMapping
}

func genCodeTypeFunc(atomicCodeMapping *atomic.Value) CodeTypeFunc {
	return func(code, service, method string) *Code {
		if v, ok := atomicCodeMapping.Load().(map[string][]*Code)[code]; ok {
			for _, vv := range v {
				if vv.Service == "" && vv.Method == "" {
					return vv
				}
				if vv.Service == service && vv.Method == "" {
					return vv
				}
				if vv.Service == "" && vv.Method == method {
					return vv
				}
				if vv.Service == service && vv.Method == method {
					return vv
				}
			}
		}
		return defaultCodeTypeFunc(code, service, method)
	}
}

func genConfigApplyFunc(cfg Config, atomicCodeMapping *atomic.Value) remote.ConfigApplyFunc {
	return func(config *operation.Operation) error {
		// codeMapping
		codeMapping := covertCodesToMapping(cfg.Codes)
		for _, v := range config.GetMetric().GetCodes() {
			codeStr := strconv.FormatInt(int64(v.GetCode()), 10)
			desc := NewCode(codeStr, CodeType(v.GetType()), v.GetDescription())
			desc.Service = v.GetService()
			desc.Method = v.GetMethod()
			codeMapping[codeStr] = append(codeMapping[codeStr], desc)
		}
		atomicCodeMapping.Store(codeMapping)
		// sever owner
		if len(config.GetOwners()) > 0 {
			var owners []string
			for _, v := range config.GetOwners() {
				owners = append(owners, v.GetName())
			}
			serverOwner := strings.Join(owners, ";")
			serverMetadata.Reset()
			serverMetadata.WithLabelValues(serverOwner, cfg.CmdbID).Set(1)
		}
		return nil
	}
}
