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
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// json方式序列化为string相比pb序列化为string有如下优点:
//
// 1. 包体包含中文可以正常显示可读性好的中文字符串而不是\346\267\261这样的以`%03o`格式打印原始的rune, 便于查看.
// 2. 便于复制数据提供给trpc cli作为请求包体, 便于调试.
// 3. 使用jsoniter序列化为json string时性能高于proto.MarshalText.
//

// ProtoMessageToJSONBytes 返回bytes, 通过log函数的%s格式化控制码转为string
func ProtoMessageToJSONBytes(message interface{}) []byte {
	out, err := jsoniter.ConfigFastest.Marshal(message)
	if err == nil {
		return out
	}
	// unexpected
	return []byte("")
}

// ProtoMessageToJSONString
func ProtoMessageToJSONString(message interface{}) string {
	out, err := jsoniter.ConfigFastest.MarshalToString(message)
	if err == nil {
		return out
	}
	// unexpected
	return ""
}

// ProtoMessageToJSONIndentBytes 多行的带缩进的pretty string bytes, 使用%s格式化控制码转为string, 相比返回string可减少一次bytes到string的转换
func ProtoMessageToJSONIndentBytes(message interface{}) []byte {
	out, err := jsoniter.ConfigFastest.MarshalIndent(message, "", "  ")
	if err == nil {
		return out
	}
	// unexpected
	return []byte("")
}

// ProtoMessageToJSONIndentString 多行的带缩进的pretty string
func ProtoMessageToJSONIndentString(message interface{}) string {
	out, err := jsoniter.ConfigFastest.MarshalIndent(message, "", "  ")
	if err == nil {
		return string(out)
	}
	// unexpected
	return ""
}

// ProtoMessageToPBJSONString 使用pbjson序列化，自动将uint64转为string
func ProtoMessageToPBJSONString(message interface{}) string {
	if p, ok := message.(proto.Message); ok {
		if out, err := protojson.Marshal(p); err == nil {
			return string(out)
		}
	}
	// unexpected
	return ""
}

// ProtoMessageToCustomJSONString 使用定制json iterator的序列化方式；int64/uint64转为string
func ProtoMessageToCustomJSONString(message interface{}) string {
	out, err := Integer64AsStringConfig.MarshalToString(message)
	if err == nil {
		return out
	}
	// unexpected
	return ""
}
