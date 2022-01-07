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

// Package api 配置组件
package api

import "go.opentelemetry.io/otel/attribute"

const (
	TpsTenantIDKey   = attribute.Key("tps.tenant.id")
	TpsTenantNameKey = attribute.Key("tps.tenant.name")
	TpsDyeingKey     = attribute.Key("tps.dyeing")
	TpsOwnerKey      = attribute.Key("server.owner")
	TpsCmdbIDKey     = attribute.Key("cmdb.module.id")

	TpsTelemetryName = "tpstelemetry"
	TenantHeaderKey  = "X-Tps-TenantID"
)
