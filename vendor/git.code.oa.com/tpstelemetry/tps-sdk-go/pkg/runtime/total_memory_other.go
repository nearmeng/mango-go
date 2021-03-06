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

//go:build !linux
// +build !linux

// Package runtime 运行时
package runtime

import (
	"fmt"
)

var (
	errTotalMemoryNotAvailable = fmt.Errorf("reading cgroups total memory is available only on linux")
	errUsageMemoryNotAvailable = fmt.Errorf("reading cgroups usage memory is available only on linux")
)

// MemoryQuota returns total available memory.
// This is non-Linux version that returns -1 and errTotalMemoryNotAvailable.
func MemoryQuota() (int64, error) {
	return -1, errTotalMemoryNotAvailable
}

// MemoryUsage returns usage memory.
// This is non-Linux version that returns -1 and errUsageMemoryNotAvailable.
func MemoryUsage() (int64, error) {
	return -1, errUsageMemoryNotAvailable
}
