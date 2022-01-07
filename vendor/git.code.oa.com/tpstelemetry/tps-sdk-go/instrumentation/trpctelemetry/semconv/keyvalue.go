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

// Package semconv 语义转换
package semconv

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/semconv"

	"git.code.oa.com/trpc-go/trpc-go"

	tpsapi "git.code.oa.com/tpstelemetry/tps-sdk-go/api"
)

const (
	NamespaceKey = attribute.Key("trpc.namespace")
	EnvNameKey   = attribute.Key("trpc.envname")

	StatusCode = attribute.Key("trpc.status_code")
	StatusMsg  = attribute.Key("trpc.status_msg")
	StatusType = attribute.Key("trpc.status_type")

	ProtocolKey = attribute.Key("trpc.protocol")

	// caller 调用 callee
	CallerServiceKey = attribute.Key("trpc.caller_service")
	CallerMethodKey  = attribute.Key("trpc.caller_method")
	CalleeServiceKey = attribute.Key("trpc.callee_service")
	CalleeMethodKey  = attribute.Key("trpc.callee_method")
)

var once sync.Once
var serviceProtocols map[string]string

func getProtocol(ctx context.Context) string {
	once.Do(func() {
		serviceProtocols = make(map[string]string)
		for _, service := range trpc.GlobalConfig().Server.Service {
			serviceProtocols[service.Name] = service.Protocol
		}
	})
	if serviceProtocols == nil {
		return ""
	}
	msg := trpc.Message(ctx)
	protocol, ok := serviceProtocols[msg.CalleeServiceName()]
	if !ok {
		return ""
	}
	return protocol
}

// KeyValues 默认 trpc 属性信息
func KeyValues(ctx context.Context) []attribute.KeyValue {
	msg := trpc.Message(ctx)
	var kvs []attribute.KeyValue
	kvs = append(kvs, NamespaceKey.String(msg.Namespace()))
	kvs = append(kvs, EnvNameKey.String(msg.EnvName()))
	kvs = append(kvs, CallerServiceKey.String(msg.CallerServiceName()))
	kvs = append(kvs, CallerMethodKey.String(msg.CallerMethod()))
	kvs = append(kvs, CalleeServiceKey.String(msg.CalleeServiceName()))
	kvs = append(kvs, CalleeMethodKey.String(msg.CalleeMethod()))
	kvs = append(kvs, tpsapi.TpsDyeingKey.String(msg.DyeingKey()))
	kvs = append(kvs, ProtocolKey.String(getProtocol(ctx)))
	kvs = append(kvs, semconv.EnduserIDKey.String(DefaultUserIDInjectFunc(ctx)))
	return kvs
}

// DefaultUserIDInjectFunc 默认trpc染色处理逻辑 handler
var DefaultUserIDInjectFunc = func(ctx context.Context) string {
	return trpc.Message(ctx).DyeingKey()
}
