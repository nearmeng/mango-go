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

import "strings"

type Level string

const (
	// TraceLevel A fine-grained debugging event. Typically disabled in default configurations.
	TraceLevel Level = "TRACE"

	// DebugLevel A debugging event.
	DebugLevel Level = "DEBUG"

	// InfoLevel An informational event. Indicates that an event happened.
	InfoLevel Level = "INFO"

	// WarnLevel A warning event. Not an error but is likely more important than an informational event.
	WarnLevel Level = "WARN"

	// ErrorLevel An error event. Something went wrong.
	ErrorLevel Level = "ERROR"

	// FatalLevel A fatal error such as application or system crash.
	FatalLevel Level = "FATAL"
)

// UnmarshalText 解析 Level
func (m *Level) UnmarshalText(text []byte) error {
	*m = Level(strings.ToUpper(string(text)))
	return nil
}
