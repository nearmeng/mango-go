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

// Package env 提供应用运行环境信息
package env

import (
	"errors"
)

// ServerInfo 服务信息
type ServerInfo struct {
	Owner  string
	CmdbID string
}

// GetServerInfo 根据环境获取服务信息
func GetServerInfo() (*ServerInfo, error) {
	if is123() {
		return get123Info()
	}
	if isStke() {
		return getStkeInfo()
	}
	return nil, errors.New("unknown env")
}
