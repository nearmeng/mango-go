// Copyright 2021 The TpsTelemetry Authors
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

//go:build linux
// +build linux

// Package runtime 运行时
package runtime

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

const (
	procSelfCgroup = "/proc/self/cgroup"
	dockerEnvPath  = "/.dockerenv"
)

var isContainer bool

var dockerQueryOnce sync.Once

// ProcessInContainer returns if process runs in container
func ProcessInContainer() bool {
	dockerQueryOnce.Do(func() {
		isContainer = hasDockerEnvPath() || hasContainerCgroups()
	})
	return isContainer
}

func hasDockerEnvPath() bool {
	_, err := os.Stat(dockerEnvPath)
	return err == nil
}

func hasContainerCgroups() bool {
	if bdata, err := ioutil.ReadFile(procSelfCgroup); err == nil {
		return strings.Contains(string(bdata), ":/docker/")
	}
	return false
}
