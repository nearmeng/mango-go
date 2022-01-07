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
	"git.code.oa.com/tpstelemetry/cgroups"

	procmeminfo "github.com/guillermo/go.procmeminfo"
)

// MemoryQuota returns total available memory.
// This implementation is meant for linux
func MemoryQuota() (int64, error) {
	if !ProcessInContainer() {
		// not in container
		memInfo := &procmeminfo.MemInfo{}
		if err := memInfo.Update(); err != nil {
			return 0, err
		}
		return int64(memInfo.Total()), nil
	}
	// uses cgroups to determine available memory.
	cgroups, err := cgroups.NewCGroupsForCurrentProcess()
	if err != nil {
		return 0, err
	}
	memoryQuota, defined, err := cgroups.MemoryQuota()
	if err != nil || !defined {
		return 0, err
	}
	return memoryQuota, nil
}

// MemoryUsage returns usage memory.
// This implementation is meant for linux
func MemoryUsage() (int64, error) {
	if !ProcessInContainer() {
		memInfo := &procmeminfo.MemInfo{}
		if err := memInfo.Update(); err != nil {
			return 0, err
		}
		return int64(memInfo.Used()), nil
	}
	// uses cgroups to determine available memory.
	cgroups, err := cgroups.NewCGroupsForCurrentProcess()
	if err != nil {
		return 0, err
	}
	memoryUsage, defined, err := cgroups.MemoryUsage()
	if err != nil || !defined {
		return 0, err
	}
	return memoryUsage, nil
}
