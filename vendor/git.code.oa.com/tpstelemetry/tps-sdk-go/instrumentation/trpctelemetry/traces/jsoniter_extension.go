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
	"reflect"
	"strconv"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
)

var Integer64AsStringConfig = jsoniter.Config{
	EscapeHTML:                    false,
	MarshalFloatWith6Digits:       true, // will lose precession
	ObjectFieldMustBeSimpleString: true, // do not unescape object field
}.Froze()

func init() {
	Integer64AsStringConfig.RegisterExtension(&integer64AsStringExtension{})
}

type wrapCodec struct {
	encodeFunc func(ptr unsafe.Pointer, stream *jsoniter.Stream)
}

// Encode Encode 接口实现
func (codec *wrapCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	codec.encodeFunc(ptr, stream)
}

// IsEmpty IsEmpty 接口实现
func (codec *wrapCodec) IsEmpty(ptr unsafe.Pointer) bool {
	return false
}

type integer64AsStringExtension struct {
	jsoniter.DummyExtension
}

// CreateEncoder 生成 encoder
func (e *integer64AsStringExtension) CreateEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	if typ.Kind() == reflect.Int64 {
		return &wrapCodec{
			encodeFunc: func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
				stream.WriteString(strconv.FormatInt(*(*int64)(ptr), 10))
			},
		}
	}

	if typ.Kind() == reflect.Uint64 {
		return &wrapCodec{
			encodeFunc: func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
				stream.WriteString(strconv.FormatUint(*(*uint64)(ptr), 10))
			},
		}
	}

	return nil
}
