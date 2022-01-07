package restful

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func init() {
	RegisterSerializer(&JSONPBSerializer{})
}

// JSONPBSerializer 用于支持 content-Type: application/json
// 基于 google.golang.org/protobuf/encoding/protojson
type JSONPBSerializer struct{}

// JSONAPI 使用 github.com/json-iterator/go 代替原生 json 库
var JSONAPI = jsoniter.ConfigCompatibleWithStandardLibrary

// Marshaller protojson 序列化结构体，可自行设置参数
var Marshaller = protojson.MarshalOptions{EmitUnpopulated: true}

// Unmarshaller protojson 反序列化结构体，可自行设置参数
var Unmarshaller = protojson.UnmarshalOptions{DiscardUnknown: true}

// Marshal 实现 Serializer
// 和 tRPC Serializer 不一样的是 RESTful API 会 marshal tRPC message 的其中一个字段
func (*JSONPBSerializer) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok { // marshal tRPC message 的其中一个字段
		return marshal(v)
	}
	// marshal tRPC message
	return Marshaller.Marshal(msg)
}

// marshal marshal tRPC message 的其中一个字段，helper 函数
func marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok { // marshal 非 proto 类型字段
		return marshalNonProtoField(v)
	}
	// marshal proto 类型字段
	return Marshaller.Marshal(msg)
}

// 包一层，用于获取 enum 名称
type wrappedEnum interface {
	protoreflect.Enum
	String() string
}

// 避免多次反射，用于判断是否实现 proto.Message 接口
var typeOfProtoMessage = reflect.TypeOf((*proto.Message)(nil)).Elem()

// marshalNonProtoField marshal 非 proto 字段
// 原生 json 库和 github.com/json-iterator/go 不支持 protobuf 的一些复杂类型
// 所以这里需要使用反射额外支持这些类型
// TODO: 性能优化
func marshalNonProtoField(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}

	// 接口值类型
	rv := reflect.ValueOf(v)

	// 指针取值
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return []byte("null"), nil
		}
		rv = rv.Elem()
	}

	// 支持 marshal enum 名称而非值
	if enum, ok := rv.Interface().(wrappedEnum); ok && !Marshaller.UseEnumNumbers {
		return JSONAPI.Marshal(enum.String())
	}
	// 支持 marshal protobuf map 类型
	if rv.Kind() == reflect.Map {
		// 构造用来 marshal 的 map
		m := make(map[string]*jsoniter.RawMessage)
		for _, key := range rv.MapKeys() { // 获取所有的 key
			// marshal value 值
			out, err := marshal(rv.MapIndex(key).Interface())
			if err != nil {
				return out, err
			}
			// 赋值
			m[fmt.Sprintf("%v", key.Interface())] = (*jsoniter.RawMessage)(&out)
			if Marshaller.Indent != "" { // 指定 indent
				return JSONAPI.MarshalIndent(v, "", Marshaller.Indent)
			}
			return JSONAPI.Marshal(v)
		}
	}
	// 支持 proto message 切片类型
	if rv.Kind() == reflect.Slice {
		if rv.IsNil() { // nil 切片
			if Marshaller.EmitUnpopulated {
				return []byte("[]"), nil
			}
			return []byte("null"), nil
		}

		if rv.Type().Elem().Implements(typeOfProtoMessage) { // proto 类型
			var buf bytes.Buffer
			buf.WriteByte('[')
			for i := 0; i < rv.Len(); i++ { // 逐个 marshal
				out, err := marshal(rv.Index(i).Interface().(proto.Message))
				if err != nil {
					return nil, err
				}
				buf.Write(out)
				if i != rv.Len()-1 {
					buf.WriteByte(',')
				}
			}
			buf.WriteByte(']')
			return buf.Bytes(), nil
		}
	}

	return JSONAPI.Marshal(v)
}

// Unmarshal 实现 Serializer
func (*JSONPBSerializer) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok { // unmarshal 到 tRPC message 的其中一个字段
		return unmarshal(data, v)
	}
	// unmarshal 到 tRPC message
	return Unmarshaller.Unmarshal(data, msg)
}

// unmarshal unmarshal 到 tRPC message 的其中一个字段
func unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok { // unmarshal 到非 proto 类型字段
		return unmarshalNonProtoField(data, v)
	}
	// unmarshal 到 proto 类型字段
	return Unmarshaller.Unmarshal(data, msg)
}

// unmarshalNonProtoField unmarshal 到非 proto 类型字段
// TODO: 性能优化
func unmarshalNonProtoField(data []byte, v interface{}) error {
	rv := reflect.ValueOf(v)

	// 必须是 ptr 类型
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("%T is not a pointer", v)
	}

	// 指针取值
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() { // nil 的话，new 一个对象
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		// 如果是 proto 类型，不用取值，直接 unmarshal
		if msg, ok := rv.Interface().(proto.Message); ok {
			return Unmarshaller.Unmarshal(data, msg)
		}
		rv = rv.Elem()
	}

	// 只支持 unmarshal 到数字型 enum
	if _, ok := rv.Interface().(wrappedEnum); ok {
		var x interface{}
		if err := jsoniter.Unmarshal(data, &x); err != nil {
			return err
		}
		switch t := x.(type) {
		case float64:
			rv.Set(reflect.ValueOf(int32(t)).Convert(rv.Type()))
			return nil
		default:
			return fmt.Errorf("unmarshalling of %T into %T is not supported", t, rv.Interface())
		}
	}
	// 支持 unmarshal 到切片
	if rv.Kind() == reflect.Slice {
		// 先 unmarshal 到 jsoniter.RawMessage 切片
		var rms []jsoniter.RawMessage
		if err := JSONAPI.Unmarshal(data, &rms); err != nil {
			return err
		}
		if rms != nil { // rv MakeSlice
			rv.Set(reflect.MakeSlice(rv.Type(), 0, 0))
		}
		// 逐个 unmarshal
		for _, rm := range rms {
			rn := reflect.New(rv.Type().Elem())
			if err := unmarshal(rm, rn.Interface()); err != nil {
				return err
			}
			rv.Set(reflect.Append(rv, rn.Elem()))
		}
		return nil
	}
	// 支持 unmarshal 到 map
	if rv.Kind() == reflect.Map {
		if rv.IsNil() { // rv MakeMap
			rv.Set(reflect.MakeMap(rv.Type()))
		}
		// 先 unmarshal 到 map[string]*jsoniter.RawMessage 中
		m := make(map[string]*jsoniter.RawMessage)
		if err := JSONAPI.Unmarshal(data, &m); err != nil {
			return err
		}
		kind := rv.Type().Key().Kind()
		for key, value := range m { // 逐对 (k, v) unmarshal
			// 转换 key
			convertedKey, err := convert(key, kind)
			if err != nil {
				return err
			}
			// unmarshal value 值
			if value == nil {
				rm := jsoniter.RawMessage("null")
				value = &rm
			}
			rn := reflect.New(rv.Type().Elem())
			if err := unmarshal([]byte(*value), rn.Interface()); err != nil {
				return err
			}
			rv.SetMapIndex(reflect.ValueOf(convertedKey), rn.Elem())
		}
	}

	return JSONAPI.Unmarshal(data, v)
}

// convert 根据 reflect.Kind 转换 map key
func convert(key string, kind reflect.Kind) (interface{}, error) {
	switch kind {
	case reflect.String:
		return key, nil
	case reflect.Bool:
		return strconv.ParseBool(key)
	case reflect.Int32:
		v, err := strconv.ParseInt(key, 0, 32)
		if err != nil {
			return nil, err
		}
		return int32(v), nil
	case reflect.Uint32:
		v, err := strconv.ParseUint(key, 0, 32)
		if err != nil {
			return nil, err
		}
		return uint32(v), nil
	case reflect.Int64:
		return strconv.ParseInt(key, 0, 64)
	case reflect.Uint64:
		return strconv.ParseUint(key, 0, 64)
	case reflect.Float32:
		v, err := strconv.ParseFloat(key, 32)
		if err != nil {
			return nil, err
		}
		return float32(v), nil
	case reflect.Float64:
		return strconv.ParseFloat(key, 64)
	default:
		return nil, fmt.Errorf("unsupported kind: %v", kind)
	}
}

// Name 实现 Serializer
func (*JSONPBSerializer) Name() string {
	return "application/json"
}

// ContentType 实现 Serializer
func (*JSONPBSerializer) ContentType() string {
	return "application/json"
}
