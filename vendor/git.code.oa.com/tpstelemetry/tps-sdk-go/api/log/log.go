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
	"context"

	"go.opentelemetry.io/otel/attribute"
)

// Trace log with trace level
func Trace(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(TraceLevel), WithFields(fields...))
}

// Debug log with debug level
func Debug(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(DebugLevel), WithFields(fields...))
}

// Info log with info level
func Info(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(InfoLevel), WithFields(fields...))
}

// Warn log with warn level
func Warn(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(WarnLevel), WithFields(fields...))
}

// Error log with error level
func Error(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(ErrorLevel), WithFields(fields...))
}

// Fatal log with fatal level
func Fatal(ctx context.Context, msg string, fields ...attribute.KeyValue) {
	logger.Log(ctx, msg, WithLevel(FatalLevel), WithFields(fields...))
}
