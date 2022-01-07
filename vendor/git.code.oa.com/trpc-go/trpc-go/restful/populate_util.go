package restful

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	// ErrTraverseNotFound 遍历 proto message 未找到字段
	ErrTraverseNotFound = errors.New("field not found")
)

// PopulateMessage 填充 proto message
func PopulateMessage(msg proto.Message, fieldPath []string, values []string) error {
	// 判空
	if len(fieldPath) == 0 || len(values) == 0 {
		return fmt.Errorf("fieldPath: %v or values: %v is empty", fieldPath, values)
	}

	// proto 反射
	message := msg.ProtoReflect()

	// 递归遍历获取字段 descriptor
	message, fd, err := traverse(message, fieldPath)
	if err != nil {
		return fmt.Errorf("failed to traverse for leaf field by fieldPath %v: %w", fieldPath, err)
	}

	// 填充字段
	switch {
	case fd.IsList(): // repeated 类型
		return populateRepeatedField(fd, message.Mutable(fd).List(), values)
	case fd.IsMap(): // map 类型
		return populateMapField(fd, message.Mutable(fd).Map(), values)
	default: // 普通类型
		return populateField(fd, message, values)
	}
}

// fdByName 根据 field name 获取 field descriptor
func fdByName(message protoreflect.Message, name string) (protoreflect.FieldDescriptor, error) {
	if message == nil {
		return nil, errors.New("get field descriptor from nil message")
	}

	field := message.Descriptor().Fields().ByJSONName(name)
	if field == nil {
		field = message.Descriptor().Fields().ByName(protoreflect.Name(name))
	}
	if field == nil {
		return nil, fmt.Errorf("%w: %v", ErrTraverseNotFound, name)
	}
	return field, nil
}

// traverse 根据 names 遍历嵌套 proto message，获取最里层的 proto message 的叶子字段 descriptor
func traverse(
	message protoreflect.Message,
	fieldPath []string,
) (protoreflect.Message, protoreflect.FieldDescriptor, error) {
	field, err := fdByName(message, fieldPath[0])
	if err != nil {
		return nil, nil, err
	}

	// 叶子字段
	if len(fieldPath) == 1 {
		return message, field, nil
	}

	// 没到叶子字段，还要继续遍历，则字段必须是 proto message 类型
	if field.Message() == nil || field.Cardinality() == protoreflect.Repeated {
		return nil, nil, fmt.Errorf("type of field %s is not proto message", fieldPath[0])
	}

	// 递归
	return traverse(message.Mutable(field).Message(), fieldPath[1:])
}

// populateField 填充普通字段
func populateField(fd protoreflect.FieldDescriptor, msg protoreflect.Message, values []string) error {
	// values 长度应该为 1
	if len(values) != 1 {
		return fmt.Errorf("tried to populate field %s with values %v", fd.FullName().Name(), values)
	}

	// 把 value 解析为 protoreflect.Value
	v, err := parseField(fd, values[0])
	if err != nil {
		return fmt.Errorf("failed to parse field %s: %w", fd.FullName().Name(), err)
	}

	// 填充
	msg.Set(fd, v)
	return nil
}

// populateRepeatedField 填充 repeated 类型字段
func populateRepeatedField(fd protoreflect.FieldDescriptor, list protoreflect.List, values []string) error {
	for _, value := range values {
		// 把 value 解析为 protoreflect.Value
		v, err := parseField(fd, value)
		if err != nil {
			return fmt.Errorf("failed to parse repeated field %s: %w", fd.FullName().Name(), err)
		}
		// 填充
		list.Append(v)
	}
	return nil
}

// populateMapField 填充 map 类型字段
func populateMapField(fd protoreflect.FieldDescriptor, m protoreflect.Map, values []string) error {
	// values 长度应该为 2
	if len(values) != 2 {
		return fmt.Errorf("tried to populate map field %s with values %v", fd.FullName().Name(), values)
	}

	// map key 值解析为 protoreflect.Value
	key, err := parseField(fd.MapKey(), values[0])
	if err != nil {
		return fmt.Errorf("failed to parse key of map field %s: %w", fd.FullName().Name(), err)
	}

	// map value 值解析为 protoreflect.Value
	value, err := parseField(fd.MapValue(), values[1])
	if err != nil {
		return fmt.Errorf("failed to parse value of map field %s: %w", fd.FullName().Name(), err)
	}

	// 填充
	m.Set(key.MapKey(), value)
	return nil
}

// parseField 根据 field descriptor 把 value 解析为 protoreflect.Value
func parseField(fd protoreflect.FieldDescriptor, value string) (protoreflect.Value, error) {
	switch kind := fd.Kind(); kind {
	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBool(v), nil
	case protoreflect.EnumKind:
		return parseEnumField(fd, value)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(v)), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(v), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(uint32(v)), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(v), nil
	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(float32(v)), nil
	case protoreflect.DoubleKind:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(v), nil
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(value), nil
	case protoreflect.BytesKind:
		v, err := base64.URLEncoding.DecodeString(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBytes(v), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return parseMessage(fd.Message(), value)
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported field kind: %v", kind)
	}
}

// parseEnumField 解析 enum 类型字段
func parseEnumField(fd protoreflect.FieldDescriptor, value string) (protoreflect.Value, error) {
	enum, err := protoregistry.GlobalTypes.FindEnumByName(fd.Enum().FullName())
	switch {
	case errors.Is(err, protoregistry.NotFound):
		return protoreflect.Value{}, fmt.Errorf("enum %s is not registered", fd.Enum().FullName())
	case err != nil:
		return protoreflect.Value{}, fmt.Errorf("failed to look up enum: %w", err)
	}
	v := enum.Descriptor().Values().ByName(protoreflect.Name(value))
	if v == nil {
		i, err := strconv.Atoi(value)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("%s is not a valid value", value)
		}
		v = enum.Descriptor().Values().ByNumber(protoreflect.EnumNumber(i))
		if v == nil {
			return protoreflect.Value{}, fmt.Errorf("%s is not a valid value", value)
		}
	}
	return protoreflect.ValueOfEnum(v.Number()), nil
}

// parseMessage 根据 message descriptor 把 value 解析成 protoreflect.Value
// 支持常用 google.protobuf.xxx 类型
func parseMessage(md protoreflect.MessageDescriptor, value string) (protoreflect.Value, error) {
	switch md.FullName() {
	case "google.protobuf.Timestamp":
		return parseTimestampMessage(value)
	case "google.protobuf.Duration":
		return parseDurationMessage(value)
	case "google.protobuf.DoubleValue":
		return parseDoubleValueMessage(value)
	case "google.protobuf.FloatValue":
		return parseFloatValueMessage(value)
	case "google.protobuf.Int64Value":
		return parseInt64ValueMessage(value)
	case "google.protobuf.Int32Value":
		return parseInt32ValueMessage(value)
	case "google.protobuf.UInt64Value":
		return parseUInt64ValueMessage(value)
	case "google.protobuf.UInt32Value":
		return parseUInt32ValueMessage(value)
	case "google.protobuf.BoolValue":
		return parseBoolValueMessage(value)
	case "google.protobuf.StringValue":
		sv := &wrapperspb.StringValue{Value: value}
		return protoreflect.ValueOfMessage(sv.ProtoReflect()), nil
	case "google.protobuf.BytesValue":
		return parseBytesValueMessage(value)
	case "google.protobuf.FieldMask":
		fm := &field_mask.FieldMask{}
		fm.Paths = append(fm.Paths, strings.Split(value, ",")...)
		return protoreflect.ValueOfMessage(fm.ProtoReflect()), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported message type: %s", string(md.FullName()))
	}
}

// parseTimestampMessage 解析 google.protobuf.Timestamp
func parseTimestampMessage(value string) (protoreflect.Value, error) {
	var msg proto.Message
	if value != "null" {
		t, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = timestamppb.New(t)
	}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseDurationMessage 解析 google.protobuf.Duration
func parseDurationMessage(value string) (protoreflect.Value, error) {
	var msg proto.Message
	if value != "null" {
		d, err := time.ParseDuration(value)
		if err != nil {
			return protoreflect.Value{}, err
		}
		msg = durationpb.New(d)
	}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseDoubleValueMessage 解析 google.protobuf.DoubleValue
func parseDoubleValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.DoubleValue{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseFloatValueMessage 解析 google.protobuf.FloatValue
func parseFloatValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.FloatValue{Value: float32(v)}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseInt64ValueMessage 解析 google.protobuf.Int64Value
func parseInt64ValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.Int64Value{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseInt32ValueMessage 解析 google.protobuf.Int32Value
func parseInt32ValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.Int32Value{Value: int32(v)}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseUInt64ValueMessage 解析 google.protobuf.UInt64Value
func parseUInt64ValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.UInt64Value{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseUInt32ValueMessage 解析 google.protobuf.UInt32Value
func parseUInt32ValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.UInt32Value{Value: uint32(v)}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseBoolValueMessage 解析 google.protobuf.BoolValue
func parseBoolValueMessage(value string) (protoreflect.Value, error) {
	v, err := strconv.ParseBool(value)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.BoolValue{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// parseBytesValueMessage 解析 google.protobuf.BytesValue
func parseBytesValueMessage(value string) (protoreflect.Value, error) {
	v, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return protoreflect.Value{}, err
	}
	msg := &wrapperspb.BytesValue{Value: v}
	return protoreflect.ValueOfMessage(msg.ProtoReflect()), nil
}

// getPopulatedFieldPaths 获取 proto message 中所有被填充过的 field path
func getPopulatedFieldPaths(message protoreflect.Message) ([]string, error) {
	var res []string
	if err := dfs(message, "", &res); err != nil {
		return nil, err
	}
	return res, nil
}

// dfs 深度优先算法
func dfs(message protoreflect.Message, path string, res *[]string) error {
	fields := message.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		if message.Has(fields.Get(i)) { // 这个字段被填充过了
			// new path 赋值
			var newPath string
			if path == "" {
				newPath = string(fields.Get(i).FullName().Name())
			} else {
				newPath = path + "." + string(fields.Get(i).FullName().Name())
			}

			// dfs
			if fields.Get(i).Message() != nil {
				child := message.Get(fields.Get(i)).Message()
				if err := dfs(child, newPath, res); err != nil {
					return err
				}
			} else {
				*res = append(*res, newPath)
			}
		}
	}

	return nil
}

// setFieldMask 为指定字段设置 field mask
func setFieldMask(message protoreflect.Message, fieldPath string) error {
	var partiallyUpdated protoreflect.FieldDescriptor
	var fieldMaskPaths []string

	fields := message.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd, err := fdByName(message, fieldPath)
		if err != nil {
			return err
		}
		partiallyUpdated = fd
	}

	if message.Get(partiallyUpdated).Message() != nil {
		paths, err := getPopulatedFieldPaths(message.Get(partiallyUpdated).Message())
		if err != nil {
			return err
		}
		fieldMaskPaths = paths
	}

	for i := 0; i < fields.Len(); i++ {
		if fields.Get(i).Kind() == protoreflect.MessageKind &&
			fields.Get(i).Message().FullName() == "google.protobuf.FieldMask" {
			fm := &field_mask.FieldMask{}
			fm.Paths = append(fm.Paths, fieldMaskPaths...)
			message.Set(fields.Get(i), protoreflect.ValueOfMessage(fm.ProtoReflect()))
			break
		}
	}

	return nil
}
