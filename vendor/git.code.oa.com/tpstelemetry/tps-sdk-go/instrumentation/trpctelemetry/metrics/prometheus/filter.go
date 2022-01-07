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

// Package prometheus prometheus metrics
package prometheus

import (
	"context"
	"strconv"

	"google.golang.org/protobuf/proto"

	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/codec"
	"git.code.oa.com/trpc-go/trpc-go/errs"
	"git.code.oa.com/trpc-go/trpc-go/filter"

	"git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/metric"
)

// ServerFilter with prometheus metric
func ServerFilter() filter.Filter {
	return func(ctx context.Context, req, rsp interface{}, handle filter.HandleFunc) (err error) {
		msg := trpc.Message(ctx)

		r := metric.NewServerReporter("trpc", msg.CallerServiceName(), msg.CallerMethod(),
			msg.CalleeServiceName(), msg.CalleeMethod())

		err = handle(ctx, req, rsp)

		r.Handled(ctx, DefaultGetCodeFunc(ctx, req, rsp, err))
		return err
	}
}

// ClientFilter with prometheus metric
func ClientFilter() filter.Filter {
	return func(ctx context.Context, req, rsp interface{}, handle filter.HandleFunc) (err error) {
		msg := trpc.Message(ctx)
		md := msg.ServerMetaData()
		monitorRequestSize(req, md)

		r := metric.NewClientReporter("trpc", msg.CallerServiceName(), msg.CallerMethod(),
			msg.CalleeServiceName(), msg.CalleeMethod())

		err = handle(ctx, req, rsp)

		r.Handled(ctx, DefaultGetCodeFunc(ctx, req, rsp, err))
		return err
	}
}

// GetCodeFunc 根据ctx req rsp得到新的用于prometheus监控使用的trpc_code label值
type GetCodeFunc func(ctx context.Context, req interface{}, rsp interface{}, err error) string

// DefaultGetCodeFunc 用户可以覆盖, 用于在rsp设置code但需要加入监控的场景
var DefaultGetCodeFunc GetCodeFunc = func(ctx context.Context,
	req interface{}, rsp interface{}, err error) string {
	if err != nil {
		return strconv.Itoa(errs.Code(err))
	}
	switch v := rsp.(type) {
	case interface {
		GetRetcode() int32
	}:
		return strconv.Itoa(int(v.GetRetcode()))
	case interface {
		GetRetCode() int32
	}:
		return strconv.Itoa(int(v.GetRetCode()))
	case interface {
		GetCode() int32
	}:
		return strconv.Itoa(int(v.GetCode()))
	default:
		return "0"
	}
}

// calcBodySize 计算proto request 包体大小
func calcBodySize(body interface{}) int {
	switch req := body.(type) {
	case proto.Message:
		return proto.Size(req)
	default:
		return 0
	}
}

// calcMetaDataSize 计算meta大小
func calcMetaDataSize(md codec.MetaData) int {
	if len(md) == 0 {
		return 0
	}

	size := 0
	for _, v := range md {
		size += len(v)
	}
	return size
}

// monitorRequestSize 监控请求的大小
func monitorRequestSize(req interface{}, md codec.MetaData) {
	ObserveRequestBodyBytes(calcBodySize(req))
	ObserveRequestMataDataBytes(calcMetaDataSize(md))
}
