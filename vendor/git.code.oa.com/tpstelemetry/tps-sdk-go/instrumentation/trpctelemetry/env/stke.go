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
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

const (
	envCmdbID    = "CMDB_MODULE_ID"
	labelsCmdbID = "moduleFourId"
	labelsPath   = "/etc/podinfo/labels"
)

func isStke() bool {
	return strings.HasPrefix(os.Getenv("POD_NAMESPACE"), "ns-")
}

func getStkeInfo() (*ServerInfo, error) {
	// 环境变量获取
	if server, err := getStkeInfoFromEnv(); err == nil {
		return server, nil
	}
	// labels获取
	if server, err := getStkeInfoFromLabels(); err == nil {
		return server, nil
	}
	return nil, errors.New("not found")
}

func getStkeInfoFromEnv() (*ServerInfo, error) {
	cmdbID := os.Getenv(envCmdbID)
	if cmdbID != "" {
		return &ServerInfo{
			CmdbID: cmdbID,
		}, nil
	}
	return nil, errors.New("not found")
}

func getStkeInfoFromLabels() (*ServerInfo, error) {
	data, err := ioutil.ReadFile(labelsPath)
	if err != nil {
		return nil, err
	}
	lines := bytes.Split(data, []byte{'\n'})
	for _, line := range lines {
		if strings.HasPrefix(string(line), labelsCmdbID) {
			cmdbID := strings.TrimSuffix(strings.TrimPrefix(string(line), labelsCmdbID+`="`), `"`)
			if cmdbID != "" {
				return &ServerInfo{
					CmdbID: cmdbID,
				}, nil
			}
		}
	}
	return nil, errors.New("not found")
}
