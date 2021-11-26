// Package pbsupport protobuf support used in db package.
package pbsupport

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/nearmeng/mango-go/plugin/log"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
)

var file_tcaplusservice_optionv1_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptor.MessageOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         60000,
		Name:          "tcaplusservice.tcaplus_primary_key",
		Tag:           "bytes,60000,opt,name=tcaplus_primary_key",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.MessageOptions)(nil),
		ExtensionType: ([]string)(nil),
		Field:         60001,
		Name:          "tcaplusservice.tcaplus_index",
		Tag:           "bytes,60001,rep,name=tcaplus_index",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.MessageOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         60002,
		Name:          "tcaplusservice.tcaplus_field_cipher_suite",
		Tag:           "bytes,60002,opt,name=tcaplus_field_cipher_suite",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.MessageOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         60003,
		Name:          "tcaplusservice.tcaplus_record_cipher_suite",
		Tag:           "bytes,60003,opt,name=tcaplus_record_cipher_suite",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.MessageOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         60004,
		Name:          "tcaplusservice.tcaplus_cipher_md5",
		Tag:           "bytes,60004,opt,name=tcaplus_cipher_md5",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.MessageOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         60005,
		Name:          "tcaplusservice.tcaplus_sharding_key",
		Tag:           "bytes,60005,opt,name=tcaplus_sharding_key",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.MessageOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         60006,
		Name:          "tcaplusservice.tcaplus_customattr",
		Tag:           "bytes,60006,opt,name=tcaplus_customattr",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.FieldOptions)(nil),
		ExtensionType: (*uint32)(nil),
		Field:         60000,
		Name:          "tcaplusservice.tcaplus_size",
		Tag:           "varint,60000,opt,name=tcaplus_size",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.FieldOptions)(nil),
		ExtensionType: (*string)(nil),
		Field:         60001,
		Name:          "tcaplusservice.tcaplus_desc",
		Tag:           "bytes,60001,opt,name=tcaplus_desc",
		Filename:      "tcaplusservice.optionv1.proto",
	},
	{
		ExtendedType:  (*descriptor.FieldOptions)(nil),
		ExtensionType: (*bool)(nil),
		Field:         60002,
		Name:          "tcaplusservice.tcaplus_crypto",
		Tag:           "varint,60002,opt,name=tcaplus_crypto",
		Filename:      "tcaplusservice.optionv1.proto",
	},
}

var (
	E_TcaplusPrimaryKey = &file_tcaplusservice_optionv1_proto_extTypes[0] //Tcaplus Primary Key

	_BlobKind = map[protoreflect.Kind]struct{}{
		protoreflect.BytesKind:   {},
		protoreflect.MessageKind: {},
		protoreflect.GroupKind:   {},
	}
	_IncreaseAbleKind = map[protoreflect.Kind]struct{}{
		protoreflect.Int32Kind:  {},
		protoreflect.Uint32Kind: {},
		protoreflect.Int64Kind:  {},
		protoreflect.Sint64Kind: {},
		protoreflect.Uint64Kind: {},
	}
	_marshalOptions        = &proto.MarshalOptions{}
	_unmarshalMergeOptions = &proto.UnmarshalOptions{
		Merge: true,
	}
)

// MarshalToMap recode split and marshal data to map, scalar will be marshal to string.
func MarshalToMap(msg proto.Message, fields []string) (map[string]interface{}, error) {
	rf := msg.ProtoReflect()
	desc := rf.Descriptor()
	rawStrMap := map[string]string{}
	rf.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if !IsFdMarshalToBlob(fd) {
			rawStrMap[string(fd.Name())] = marshalScalar(fd, v)
		}
		return true
	})

	fdFilter := DirtyFilter(desc, fields)
	fds := desc.Fields()
	// TODO: 这里需要优化，不能调用marshalField函数还是会进行多余的编码.
	wireMap := map[string]interface{}{}
	var cur int32
	marshalBytes, err := _marshalOptions.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal to map err:%w", err)
	}
	for int(cur) < len(marshalBytes) {
		num, _, n := protowire.ConsumeField(marshalBytes[cur:])
		if n < 0 {
			return nil, fmt.Errorf("wire consume field ret=%d", n)
		}
		next := cur + int32(n)
		fd := fds.ByNumber(num)
		if fd == nil {
			return nil, fmt.Errorf("unknow field num=%d", num)
		}
		fdName := string(fd.Name())
		if fdFilter != nil {
			if _, exist := fdFilter[fdName]; !exist {
				cur = next
				continue
			}
		}
		rstr, exist := rawStrMap[fdName]
		if exist {
			wireMap[fdName] = rstr
		} else {
			cb, exist := wireMap[fdName]
			if !exist {
				wireMap[fdName] = marshalBytes[cur:next]
			} else {
				wireMap[fdName] = append(cb.([]byte), marshalBytes[cur:next]...)
			}
		}
		cur = next
	}
	return wireMap, nil
}

// DirtyFilter Create dirty filter and ignore deep level fields.
func DirtyFilter(desc protoreflect.MessageDescriptor, fields []string) map[string]struct{} {
	if len(fields) == 0 {
		return nil
	}
	fds := desc.Fields()
	retFilter := map[string]struct{}{}
	for _, fs := range fields {
		retKey := fs
		fd := fds.ByName(protoreflect.Name(fs))
		if fd == nil {
			// TODO:need optimize.
			ir := strings.Index(fs, ".")
			if ir < 0 {
				log.Info("dirty field=%s not in msg=%s", fs, desc.FullName())
				continue
			}
			retKey = fs[:ir]
			fd = fds.ByName(protoreflect.Name(retKey))
			if fd == nil {
				log.Info("dirty field=%s not in msg=%s", fs, desc.FullName())
				continue
			}
		}
		retFilter[retKey] = struct{}{}
	}
	return retFilter
}

// UnmarshalFromMap unmarsh data loaded from db.
func UnmarshalFromMap(msg proto.Message, bytesMap map[string]string) (err error) {
	buf := bytes.Buffer{}
	rf := msg.ProtoReflect()
	desc := rf.Descriptor()
	fds := desc.Fields()
	for k, s := range bytesMap {
		if strings.HasPrefix(k, "_") {
			continue
		}
		fd := fds.ByName(protoreflect.Name(k))
		if fd == nil {
			log.Trace("cannot find field num=%d type=%s", k, rf.Descriptor().FullName())
			continue
		}
		if IsFdMarshalToBlob(fd) {
			_, e := buf.WriteString(s)
			if e != nil {
				err = fmt.Errorf("merge buf err:%w", e)
				return
			}
		} else {
			v, e1 := unmarshalScalarByStr(fd, s)
			if err != nil {
				err = e1
				return
			}
			rf.Set(fd, v)
		}
	}
	err = _unmarshalMergeOptions.Unmarshal(buf.Bytes(), msg)
	return
}

// FindPrimaryKey find primary key.
func FindPrimaryKey(desc protoreflect.Descriptor) []string {
	// TODO: 暂时使用tcaplus的option.
	primKey, ok := proto.GetExtension(desc.Options(), E_TcaplusPrimaryKey).(string)
	if !ok || len(primKey) == 0 {
		return nil
	}
	return strings.Split(primKey, ",")
}

// FindFds find field descriptor, keep order of param KeyNames.
func FindFds(msgDesc protoreflect.MessageDescriptor, keyNames []string) []protoreflect.FieldDescriptor {
	fds := make([]protoreflect.FieldDescriptor, len(keyNames))
	fields := msgDesc.Fields()
	for i, key := range keyNames {
		fds[i] = fields.ByTextName(key)
	}
	return fds
}

func unmarshalScalarByStr(fd protoreflect.FieldDescriptor, str string) (protoreflect.Value, error) {
	const b32 int = 32
	const b64 int = 64
	const base10 = 10

	kind := fd.Kind()
	switch kind {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(str), nil

	case protoreflect.BoolKind:
		switch str {
		case "true", "1":
			return protoreflect.ValueOfBool(true), nil
		case "false", "0", "":
			return protoreflect.ValueOfBool(false), nil
		}

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		if n, err := strconv.ParseInt(str, base10, b32); err == nil {
			return protoreflect.ValueOfInt32(int32(n)), nil
		}

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		if n, err := strconv.ParseInt(str, base10, b64); err == nil {
			return protoreflect.ValueOfInt64(n), nil
		}

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if n, err := strconv.ParseUint(str, base10, b32); err == nil {
			return protoreflect.ValueOfUint32(uint32(n)), nil
		}

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if n, err := strconv.ParseUint(str, base10, b64); err == nil {
			return protoreflect.ValueOfUint64(n), nil
		}
	case protoreflect.DoubleKind:
		if n, err := strconv.ParseFloat(str, b64); err == nil {
			return protoreflect.ValueOfFloat64(n), nil
		}
	case protoreflect.FloatKind:
		if n, err := strconv.ParseFloat(str, b64); err == nil {
			return protoreflect.ValueOfFloat32(float32(n)), nil
		}
	}

	return protoreflect.Value{}, fmt.Errorf("invalid value for fd=%s value=%s", fd.Name(), str)
}

func marshalScalar(fd protoreflect.FieldDescriptor, v protoreflect.Value) (retstr string) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		if v.Bool() {
			retstr = "1"
		} else {
			retstr = "0"
		}
	default:
		retstr = v.String()
	}
	return
}

// IsFdMarshalToBlob if a field will marshal to blob.
func IsFdMarshalToBlob(fd protoreflect.FieldDescriptor) bool {
	if fd.Cardinality() == protoreflect.Repeated {
		return true
	}
	_, exist := _BlobKind[fd.Kind()]
	return exist
}

// BuildKeyFieldsMap build filter of key.
func BuildKeyFieldsMap(desc protoreflect.MessageDescriptor) map[string]struct{} {
	ret := map[string]struct{}{}
	keys := FindPrimaryKey(desc)
	for _, key := range keys {
		ret[key] = struct{}{}
	}
	return ret
}

// BuildIncreaseableFieldsMap build filter of field can increase.
func BuildIncreaseableFieldsMap(desc protoreflect.MessageDescriptor) map[string]struct{} {
	ret := map[string]struct{}{}
	keys := BuildKeyFieldsMap(desc)

	fields := desc.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.Cardinality() == protoreflect.Repeated {
			continue
		}
		if _, exist := keys[string(fd.Name())]; exist {
			continue
		}
		if _, exist := _IncreaseAbleKind[fd.Kind()]; !exist {
			continue
		}
		ret[string(fd.Name())] = struct{}{}
	}
	return ret
}
