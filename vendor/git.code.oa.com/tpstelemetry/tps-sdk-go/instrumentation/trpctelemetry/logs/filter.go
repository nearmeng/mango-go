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

// Package logs 日志组件
package logs

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"
	"go.opentelemetry.io/otel/attribute"

	apilog "git.code.oa.com/tpstelemetry/tps-sdk-go/api/log"
	"git.code.oa.com/tpstelemetry/tps-sdk-go/instrumentation/trpctelemetry/semconv"
)

var (
	systemField = attribute.String("system", "trpc")
	serverField = attribute.String("span.kind", "server")
	clientField = attribute.String("span.kind", "client")
)

func decimal(value float64) float64 {
	value, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return value
}
func durationToMilliseconds(duration time.Duration) float64 {
	return decimal(float64(duration.Nanoseconds()/1000) / 1000)
}

// ServerFilter 服务端插件
func ServerFilter() filter.Filter {
	return func(ctx context.Context, req, rsp interface{}, handle filter.HandleFunc) (err error) {
		startTime := time.Now()
		var fields []attribute.KeyValue
		fields = append(fields, attribute.String("trpc.start_time", startTime.Format(time.RFC3339)))
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, attribute.String("trpc.request.deadline", d.Format(time.RFC3339)))
		}

		err = handle(ctx, req, rsp)

		code := errs.Code(err)

		fields = append(fields, semconv.KeyValues(ctx)...)
		fields = append(fields, []attribute.KeyValue{systemField, serverField}...)
		fields = append(fields, attribute.Float64("trpc.time_ms", durationToMilliseconds(time.Since(startTime))))
		fields = append(fields, attribute.Int("trpc.code", code))
		apilog.GlobalLogger().Log(ctx, "", apilog.WithLevel(apilog.InfoLevel),
			apilog.WithFields(fields...),
			apilog.WithStandaloneEnable())
		return err
	}
}

// ClientFilter 客户端插件
func ClientFilter() filter.Filter {
	return func(ctx context.Context, req interface{}, rsp interface{}, handle filter.HandleFunc) (err error) {
		fields := []attribute.KeyValue{
			systemField,
			clientField,
		}
		startTime := time.Now()
		err = handle(ctx, req, rsp)
		code := errs.Code(err)

		fields = append(fields, semconv.KeyValues(ctx)...)
		fields = append(fields, attribute.Float64("trpc.time_ms", durationToMilliseconds(time.Since(startTime))))
		fields = append(fields, attribute.Int("trpc.code", code))
		apilog.Info(ctx, "", fields...)
		return err
	}
}
