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
	"encoding/json"
	"fmt"
	"time"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/remote"
)

// Config metric config
type Config struct {
	// Enabled open or close
	Enabled bool `yaml:"enabled"`
	// RegistryEndpoints registry addrs
	RegistryEndpoints []string `yaml:"registry_endpoints"`
	// TTL Time to live
	TTL time.Duration `yaml:"ttl"`
	// Instance registry instance info
	Instance Instance `yaml:"instance"`
	// ServerOwner is server owner user, for send alert
	ServerOwner string `yaml:"server_owner"`
	// CmdbID is cmdbID
	CmdbID string `yaml:"cmdb_id"`
	// Codes code mapping
	Codes []*Code `yaml:"codes"`
	// Configurator 配置器, 支持动态配置, 实验性功能.
	Configurator remote.Configurator `yaml:"-"`
}

// Instance 业务实例
type Instance struct {
	// Addr metrics server的导出数据地址IP:PORT
	Addr string `json:"addr" yaml:"addr"`
	// TenantID 业务所属租户ID
	TenantID string `json:"tenant_id" yaml:"tenant_id"`
	// Metadata 实例相关元数据信息
	Metadata map[string]string `json:"metadata" yaml:"metadata"`
	// KeySuffix 注册到Etcd的Key后缀，不参与json序列化
	KeySuffix string `json:"-" yaml:"-"`
	// Key 注册到Etcd的Key，不参与json序列化. 完全自定义Key, 需小心使用.
	Key string `json:"-" yaml:"-"`
}

func toKey(ins *Instance) string {
	if ins == nil {
		return ""
	}
	if ins.Key != "" {
		return ins.Key
	}
	return fmt.Sprintf(
		"/tpstelemetry/metrics/services/%s/%s%s", ins.TenantID, ins.Addr, ins.KeySuffix)
}

func toValue(ins *Instance) string {
	if ins != nil {
		if data, err := json.Marshal(ins); nil == err {
			return string(data)
		}
	}
	return "{}"
}
