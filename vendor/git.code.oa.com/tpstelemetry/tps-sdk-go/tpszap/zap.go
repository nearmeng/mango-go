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

// Package tpszap zap组件适配
package tpszap

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	commonproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/common/v1"
	logsproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/logs/v1"
	resourceproto "git.code.oa.com/tpstelemetry/tpstelemetry-protocol/opentelemetry/proto/resource/v1"
	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"

	apilog "git.code.oa.com/tpstelemetry/tps-sdk-go/api/log"
	sdklog "git.code.oa.com/tpstelemetry/tps-sdk-go/sdk/log"
)

var _ zapcore.Encoder = (*encoder)(nil)

type encoder struct {
	*zapcore.EncoderConfig

	record *logsproto.LogRecord
	kvs    []*commonproto.KeyValue

	buf *buffer.Buffer
}

var bufPool = buffer.NewPool()

// NewEncoder NewEncoder
func NewEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return &encoder{
		EncoderConfig: &cfg,
		record:        &logsproto.LogRecord{},
		buf:           bufPool.Get(),
	}
}

var errUnimplemented = errors.New("errUnimplemented")

// AddArray AddArray
func (e *encoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	return errUnimplemented
}

// AddObject AddObject
func (e *encoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	return errUnimplemented
}

// AddBinary AddBinary
func (e *encoder) AddBinary(key string, value []byte) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: string(value),
			},
		},
	})
}

// AddByteString AddByteString
func (e *encoder) AddByteString(key string, value []byte) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: string(value),
			},
		},
	})
}

// AddBool AddBool
func (e *encoder) AddBool(key string, value bool) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_BoolValue{
				BoolValue: value,
			},
		},
	})
}

// AddComplex128 AddComplex128
func (e *encoder) AddComplex128(key string, value complex128) {
}

// AddComplex64 AddComplex64
func (e *encoder) AddComplex64(key string, value complex64) {
}

// AddDuration AddDuration
func (e *encoder) AddDuration(key string, value time.Duration) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: value.String(),
			},
		},
	})
}

// AddFloat64 AddFloat64
func (e *encoder) AddFloat64(key string, value float64) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_DoubleValue{
				DoubleValue: value,
			},
		},
	})
}

// AddFloat32 AddFloat32
func (e *encoder) AddFloat32(key string, value float32) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_DoubleValue{
				DoubleValue: float64(value),
			},
		},
	})
}

// AddInt AddInt
func (e *encoder) AddInt(key string, value int) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddInt64 AddInt64
func (e *encoder) AddInt64(key string, value int64) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: value,
			},
		},
	})
}

// AddInt32 AddInt32
func (e *encoder) AddInt32(key string, value int32) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddInt16 AddInt16
func (e *encoder) AddInt16(key string, value int16) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddInt8 AddInt8
func (e *encoder) AddInt8(key string, value int8) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddString AddString
func (e *encoder) AddString(key, value string) {
	if key == "sampled" {
		if value == strconv.FormatBool(true) {
			e.record.Flags = 1
		}
		return
	}
	if key == "traceID" {
		e.record.TraceId, _ = hex.DecodeString(value)
		return
	}
	if key == "spanID" {
		e.record.SpanId, _ = hex.DecodeString(value)
		return
	}
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: value,
			},
		},
	})
}

// AddTime AddTime
func (e *encoder) AddTime(key string, value time.Time) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_StringValue{
				StringValue: value.String(),
			},
		},
	})
}

// AddUint AddUint
func (e *encoder) AddUint(key string, value uint) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddUint64 AddUint64
func (e *encoder) AddUint64(key string, value uint64) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddUint32 AddUint32
func (e *encoder) AddUint32(key string, value uint32) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddUint16 AddUint16
func (e *encoder) AddUint16(key string, value uint16) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddUint8 AddUint8
func (e *encoder) AddUint8(key string, value uint8) {
	e.kvs = append(e.kvs, &commonproto.KeyValue{
		Key: key,
		Value: &commonproto.AnyValue{
			Value: &commonproto.AnyValue_IntValue{
				IntValue: int64(value),
			},
		},
	})
}

// AddUintptr AddUintptr
func (e *encoder) AddUintptr(key string, value uintptr) {

}

// AddReflected AddReflected
func (e *encoder) AddReflected(key string, value interface{}) error {
	return nil
}

// OpenNamespace OpenNamespace
func (e *encoder) OpenNamespace(key string) {
}

// Clone Clone
func (e *encoder) Clone() zapcore.Encoder {
	enc := &encoder{
		EncoderConfig: e.EncoderConfig,
		kvs:           make([]*commonproto.KeyValue, 0, len(e.kvs)),
		record:        e.record,
		buf:           e.buf,
	}
	enc.kvs = append(enc.kvs, e.kvs...)
	return enc
}

// nolint
func (e *encoder) convertField(f zapcore.Field) {
	switch f.Type {
	case zapcore.ArrayMarshalerType:
		_ = e.AddArray(f.Key, f.Interface.(zapcore.ArrayMarshaler))
	case zapcore.ObjectMarshalerType:
		_ = e.AddObject(f.Key, f.Interface.(zapcore.ObjectMarshaler))
	case zapcore.BinaryType:
		e.AddBinary(f.Key, f.Interface.([]byte))
	case zapcore.BoolType:
		e.AddBool(f.Key, f.Integer == 1)
	case zapcore.ByteStringType:
		e.AddByteString(f.Key, f.Interface.([]byte))
	case zapcore.Complex128Type:
		e.AddComplex128(f.Key, f.Interface.(complex128))
	case zapcore.Complex64Type:
		e.AddComplex64(f.Key, f.Interface.(complex64))
	case zapcore.DurationType:
		e.AddDuration(f.Key, time.Duration(f.Integer))
	case zapcore.Float64Type:
		e.AddFloat64(f.Key, math.Float64frombits(uint64(f.Integer)))
	case zapcore.Float32Type:
		e.AddFloat32(f.Key, math.Float32frombits(uint32(f.Integer)))
	case zapcore.Int64Type:
		e.AddInt64(f.Key, f.Integer)
	case zapcore.Int32Type:
		e.AddInt32(f.Key, int32(f.Integer))
	case zapcore.Int16Type:
		e.AddInt16(f.Key, int16(f.Integer))
	case zapcore.Int8Type:
		e.AddInt8(f.Key, int8(f.Integer))
	case zapcore.StringType:
		e.AddString(f.Key, f.String)
	case zapcore.TimeType:
		if f.Interface != nil {
			e.AddTime(f.Key, time.Unix(0, f.Integer).In(f.Interface.(*time.Location)))
		} else {
			// Fall back to UTC if location is nil.
			e.AddTime(f.Key, time.Unix(0, f.Integer))
		}
	case zapcore.TimeFullType:
		e.AddTime(f.Key, f.Interface.(time.Time))
	case zapcore.Uint64Type:
		e.AddUint64(f.Key, uint64(f.Integer))
	case zapcore.Uint32Type:
		e.AddUint32(f.Key, uint32(f.Integer))
	case zapcore.Uint16Type:
		e.AddUint16(f.Key, uint16(f.Integer))
	case zapcore.Uint8Type:
		e.AddUint8(f.Key, uint8(f.Integer))
	case zapcore.UintptrType:
		e.AddUintptr(f.Key, uintptr(f.Integer))
	case zapcore.ReflectType:
		_ = e.AddReflected(f.Key, f.Interface)
	case zapcore.NamespaceType:
		e.OpenNamespace(f.Key)
	case zapcore.StringerType:
	case zapcore.ErrorType:
		e.AddString(f.Key, f.Interface.(error).Error())
	case zapcore.SkipType:
		break
	default:
		panic(fmt.Sprintf("unknown field type: %v", f))
	}

}

// EncodeEntry EncodeEntry
func (e *encoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	e.record.SeverityText = entry.Level.String()
	for _, f := range fields {
		e.convertField(f)
	}
	e.record.Attributes = append(e.record.Attributes, e.kvs...)
	e.record.Body = &commonproto.AnyValue{
		Value: &commonproto.AnyValue_StringValue{
			StringValue: entry.Message,
		},
	}
	e.record.Name = entry.LoggerName
	e.record.TimeUnixNano = uint64(entry.Time.UnixNano())

	data, err := proto.Marshal(e.record)
	if err != nil {
		return nil, err
	}

	_, err = e.buf.Write(data)
	if err != nil {
		return nil, err
	}

	return e.buf, nil
}

var _ zapcore.WriteSyncer = (*writeSyncer)(nil)
var _ zapcore.WriteSyncer = (*jsonWriteSyncer)(nil)

type jsonWriteSyncer struct {
}

// Sync Sync
func (jw *jsonWriteSyncer) Sync() error {
	return nil
}

// Write Write
func (jw *jsonWriteSyncer) Write(p []byte) (n int, err error) {
	raw := make(map[string]interface{})
	err = jsoniter.ConfigFastest.Unmarshal(p, &raw)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

type writeSyncer struct {
	rs        *resource.Resource
	processor *sdklog.BatchProcessor
}

// Write Write
func (w *writeSyncer) Write(p []byte) (n int, err error) {
	record := &logsproto.LogRecord{}
	err = proto.Unmarshal(p, record)
	if err != nil {
		return 0, err
	}

	rs := &resourceproto.Resource{}
	for _, kv := range w.rs.Attributes() {
		rs.Attributes = append(rs.Attributes, &commonproto.KeyValue{
			Key: string(kv.Key),
			Value: &commonproto.AnyValue{
				Value: &commonproto.AnyValue_StringValue{StringValue: kv.Value.Emit()},
			},
		})
	}

	rl := &logsproto.ResourceLogs{
		Resource: rs,
		InstrumentationLibraryLogs: []*logsproto.InstrumentationLibraryLogs{
			{
				Logs: []*logsproto.LogRecord{record},
			},
		},
	}
	w.processor.Enqueue(rl)
	return len(p), nil
}

// Sync Sync
func (w *writeSyncer) Sync() error {
	return nil
}

// NewWriteSyncer NewWriteSyncer
func NewWriteSyncer(p *sdklog.BatchProcessor, rs *resource.Resource) zapcore.WriteSyncer {
	return &writeSyncer{
		processor: p,
		rs:        rs,
	}
}

// NewJSONWriteSyncer NewJSONWriteSyncer
func NewJSONWriteSyncer() zapcore.WriteSyncer {
	return &jsonWriteSyncer{}
}

// NewCore NewCore
func NewCore(opts ...sdklog.LoggerOption) zapcore.Core {
	o := &sdklog.LoggerOptions{
		LevelEnabled: apilog.DebugLevel,
	}
	for _, opt := range opts {
		opt(o)
	}
	return zapcore.NewCore(NewEncoder(zap.NewProductionEncoderConfig()),
		NewWriteSyncer(o.Processor, o.Resource), toLevelEnabler(o.LevelEnabled))
}

// NewBatchCore NewBatchCore
func NewBatchCore(syncer *BatchWriteSyncer, opts ...sdklog.LoggerOption) zapcore.Core {
	o := &sdklog.LoggerOptions{
		LevelEnabled: apilog.DebugLevel,
	}
	for _, opt := range opts {
		opt(o)
	}
	return zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig()),
		syncer, toLevelEnabler(o.LevelEnabled))
}

func encoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeCaller = zapcore.FullCallerEncoder
	return cfg
}

// NewJSONCore NewJSONCore
func NewJSONCore() zapcore.Core {
	return zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		NewJSONWriteSyncer(), zapcore.DebugLevel)
}

func toLevelEnabler(level apilog.Level) zapcore.LevelEnabler {
	switch level {
	case apilog.TraceLevel:
		return zap.DebugLevel
	case apilog.DebugLevel:
		return zap.DebugLevel
	case apilog.InfoLevel:
		return zap.InfoLevel
	case apilog.WarnLevel:
		return zap.WarnLevel
	case apilog.ErrorLevel:
		return zap.ErrorLevel
	case apilog.FatalLevel:
		return zap.FatalLevel
	default:
		return zap.ErrorLevel
	}
}
