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

// Package log 天机阁log接口
package log

import (
	"go.opentelemetry.io/otel/attribute"
)

// Config 日志配置信息
type Config struct {
	Level             Level
	Name              string
	Fields            []attribute.KeyValue
	StandaloneEnabled bool
}

// Option 配置控制func
type Option func(*Config)

// WithLevel 指定级别
func WithLevel(lvl Level) Option {
	return func(c *Config) {
		c.Level = lvl
	}
}

// WithStandaloneEnable 指定本地
func WithStandaloneEnable() Option {
	return func(c *Config) {
		c.StandaloneEnabled = true
	}
}

// WithName 指定名字
func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

// WithFields 指定属性字段
func WithFields(fields ...attribute.KeyValue) Option {
	return func(c *Config) {
		c.Fields = fields
	}
}
