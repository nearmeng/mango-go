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

// Package traces trpc traces 组件
package traces

import (
	"context"

	"git.code.oa.com/trpc-go/trpc-go/log"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/config"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/logs"
)

// doFlowLog
func doFlowLog(ctx context.Context, flow *logs.FlowLog, options FilterOptions) {
	switch options.TraceLogMode {
	case config.LogModeDisable:
		return
	case config.LogModeMultiLine:
		log.DebugContextf(ctx, "%s", flow.MultilineString())
		return
	case config.LogModeOneLine, config.LogModeDefault:
	default:
	}
	log.DebugContextf(ctx, "%s", flow.OneLineString())
}
