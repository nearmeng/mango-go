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
	"strings"
	"time"

	tkafka "git.code.oa.com/trpc-go/trpc-database/kafka"
	"git.code.oa.com/trpc-go/trpc-go/plugin"

	"github.com/Shopify/sarama"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
)

const (
	pluginType = "telemetry"
)

var _ plugin.Factory = (*factory)(nil)

type factory struct {
}

// Type Type 方法
func (f factory) Type() string {
	return pluginType
}

// Setup Setup方法
func (f factory) Setup(name string, configDec plugin.Decoder) error {
	return nil
}

// addEvent 返回messageStr以便后续的处理可以复用减少序列化消耗, 上层需要判空, 如果为空,
// 则表示包体不是proto.Message, 没有进行序列化为string
func addEvent(ctx context.Context, message interface{},
	messageType attribute.KeyValue) (messageStr string) {
	var t time.Duration
	deadline, ok := ctx.Deadline()
	if ok {
		t = time.Until(deadline)
	}
	span := trace.SpanFromContext(ctx)
	messageStr = fixStringTooLong(DefaulTraceEventMsgMarshaler(message))
	span.AddEvent(messageType.Value.AsString(),
		trace.WithAttributes(
			// RPCMessageUncompressedSizeKey 非准确值,
			// 但比pb1.4.0之前隐含一次序列化的proto.Size消耗小得多
			semconv.RPCMessageUncompressedSizeKey.Int(len(messageStr)),
			attribute.Key("message.detail").String(messageStr),
			attribute.Key("ctx.deadline").String(t.String()),
		),
	)
	return messageStr
}

const fixedStringSuffix = "...stringLengthTooLong"
const defaultMaxStringLength = 32766

var maxStringLength = defaultMaxStringLength

// SetMaxStringLength 允许用户自己调整最大长度
func SetMaxStringLength(limit int) {
	if limit > defaultMaxStringLength {
		return
	}
	maxStringLength = limit
}

// isStringTooLong
func isStringTooLong(s string) bool {
	return len(s) > maxStringLength
}

// fixStringTooLong
// Document contains at least one immense term in field=\"logs.fields.value\"
// (whose UTF8 encoding is longer than the max length 32766)
func fixStringTooLong(s string) (result string) {
	if isStringTooLong(s) {
		return strings.ToValidUTF8(s[:maxStringLength-len(fixedStringSuffix)]+fixedStringSuffix, "")
	}
	return s
}

const (
	// w3c trace header 字段
	traceparentHeader = "traceparent"
	tracestateHeader  = "tracestate"
	// baggageHeader 业务方自定义的全链路感知信息
	baggageHeader = "baggage"
)

// InjectTraceContextToKafkaHead 向kafka ProducerMessage head 中注入trace 信息
func InjectTraceContextToKafkaHead(kafkaReq *tkafka.Request, sup *supplier) {
	if sup == nil {
		return
	}
	// traceParent 为空则无需注入
	traceParent := sup.Get(traceparentHeader)
	if traceParent == "" {
		return
	}
	// traceState 目前属于保留字段
	traceState := sup.Get(tracestateHeader)
	// baggage 业务方自定义的全链路感知信息
	baggage := sup.Get(baggageHeader)
	headers := kafkaReq.Headers
	// 保证key的唯一性
	for i := 0; i < len(headers); i++ {
		if string(headers[i].Key) == traceparentHeader || string(headers[i].Key) == tracestateHeader ||
			string(headers[i].Key) == baggageHeader {
			headers = append(headers[:i], headers[i+1:]...)
			i--
		}
	}
	headers = append(headers, sarama.RecordHeader{
		Key:   []byte(traceparentHeader),
		Value: []byte(traceParent),
	}, sarama.RecordHeader{
		Key:   []byte(tracestateHeader),
		Value: []byte(traceState),
	}, sarama.RecordHeader{
		Key:   []byte(baggageHeader),
		Value: []byte(baggage),
	})
	kafkaReq.Headers = headers
}

// ExtractTraceContextFromKafkaHead 从ConsumerMessage header 中提取trace信息
func ExtractTraceContextFromKafkaHead(msg *sarama.ConsumerMessage) ([]byte, []byte, []byte) {
	var traceParent, traceState, baggage []byte
	for _, h := range msg.Headers {
		if h != nil && string(h.Key) == traceparentHeader {
			traceParent = h.Value
		}
		if h != nil && string(h.Key) == tracestateHeader {
			traceState = h.Value
		}
		if h != nil && string(h.Key) == baggageHeader {
			baggage = h.Value
		}
	}
	return traceParent, traceState, baggage
}
