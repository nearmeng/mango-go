// Package protolog is a configurable text format marshaler for log.
package protolog

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MarshalOptions is a configurable text format marshaler.
type MarshalOptions struct{}

type encoder struct {
	builder *strings.Builder
}

const (
	_DefaultEncodeCap = 512
)

// Format formats the message as a string.
// This method is only intended for human consumption and ignores errors.
// Do not depend on the output being stable. It may change over time across
// different versions of the program.
func (o MarshalOptions) Format(m proto.Message) string {
	if m == nil || !m.ProtoReflect().IsValid() {
		return "nil" // invalid syntax, but okay since this is for debugging
	}
	b, _ := o.Marshal(m)
	return b
}

// Marshal a pb msg to log format.
func (o MarshalOptions) Marshal(m proto.Message) (ret string, err error) {
	rf := m.ProtoReflect()
	builder := &strings.Builder{}
	builder.Grow(_DefaultEncodeCap)
	enc := encoder{builder: builder}
	_, _ = builder.WriteString("message ")
	if err = enc.marshalMessage(rf, nil); err != nil {
		return
	}
	ret = builder.String()
	return
}

func (e encoder) marshalMessage(rf protoreflect.Message, path []string) (err error) {
	var i int
	rf.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if od := fd.ContainingOneof(); od != nil {
			fd = rf.WhichOneof(od)
		}
		if i > 0 {
			_ = e.builder.WriteByte(' ')
		}
		i++
		err = e.marshalField(rf.Get(fd), fd, path)
		return err == nil
	})
	return nil
}

// marshalField marshals the given field with protoreflect.Value.
func (e encoder) marshalField(val protoreflect.Value, fd protoreflect.FieldDescriptor, path []string) error {
	switch {
	case fd.IsList():
		return e.marshalList(val.List(), fd, path)
	case fd.IsMap():
		return e.marshalMap(val.Map(), fd, path)
	default:
		return e.marshalSingular(val, fd, path)
	}
}

func (e encoder) marshalList(list protoreflect.List, fd protoreflect.FieldDescriptor, path []string) error {
	path = append(path, fdNameToStr(fd))
	for i := 0; i < list.Len(); i++ {
		if i > 0 {
			_ = e.builder.WriteByte(' ')
		}
		fpath := path
		fpath = append(fpath, strconv.Itoa(i))
		if err := e.marshalListOrMapSingular(list.Get(i), fd, fpath); err != nil {
			return err
		}
	}
	return nil
}

// marshalMap marshals the given protoreflect.Map as multiple name-value fields.
func (e encoder) marshalMap(mmap protoreflect.Map, fd protoreflect.FieldDescriptor, path []string) error {
	var err error
	path = append(path, fdNameToStr(fd))
	var i int
	mmap.Range(func(mk protoreflect.MapKey, v protoreflect.Value) bool {
		if i > 0 {
			_ = e.builder.WriteByte(' ')
		}
		i++
		var keyStr string
		keyStr, err = valueToString(mk.Value(), fd.MapKey())
		if err != nil {
			return false
		}
		if err = e.marshalListOrMapSingular(v, fd.MapValue(), append(path, keyStr)); err != nil {
			return false
		}
		return true
	})
	return err
}

func (e encoder) marshalListOrMapSingular(val protoreflect.Value, fd protoreflect.FieldDescriptor, path []string) error {
	if fd.Kind() == protoreflect.MessageKind {
		return e.marshalMessage(val.Message(), append(path, fdNameToStr(fd)))
	}
	retStr, err := valueToString(val, fd)
	if err != nil {
		return err
	}
	sb := e.builder
	for i, p := range path {
		_, _ = sb.WriteString(p)
		if i < len(path)-1 {
			_ = sb.WriteByte('.')
		}
	}
	_ = sb.WriteByte('=')
	_, _ = sb.WriteString(retStr)
	return nil
}

func (e encoder) marshalSingular(val protoreflect.Value, fd protoreflect.FieldDescriptor, path []string) error {
	if fd.Kind() == protoreflect.MessageKind {
		return e.marshalMessage(val.Message(), append(path, fdNameToStr(fd)))
	}
	retStr, err := valueToString(val, fd)
	if err != nil {
		return err
	}
	sb := e.builder
	for _, p := range path {
		_, _ = sb.WriteString(p)
		_ = sb.WriteByte('.')
	}
	_, _ = sb.WriteString(fdNameToStr(fd))
	_ = sb.WriteByte('=')
	_, _ = sb.WriteString(retStr)
	return nil
}

// fdNameToStr some special log key need fix.
func fdNameToStr(fd protoreflect.FieldDescriptor) string {
	fn := fd.Name()
	switch fn {
	case "MsgID":
		return "msg"
	case "Uid", "UID":
		return "uid"
	default:
		return string(fn)
	}
}

func valueToString(val protoreflect.Value, fd protoreflect.FieldDescriptor) (retStr string, err error) {
	const (
		b10 = 10
		b32 = 64
		b64 = 64
	)

	kind := fd.Kind()
	switch kind {
	case protoreflect.BoolKind:
		if val.Bool() {
			retStr = "true"
		} else {
			retStr = "false"
		}

	case protoreflect.StringKind:
		retStr = val.String()

	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		retStr = strconv.FormatInt(val.Int(), b10)

	case protoreflect.Uint64Kind:
		retStr = strconv.FormatUint(val.Uint(), b10)

	case protoreflect.Uint32Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		retStr = strconv.FormatUint(val.Uint(), b10)

	case protoreflect.FloatKind:
		retStr = strconv.FormatFloat(val.Float(), 'f', b10, b32)

	case protoreflect.DoubleKind:
		retStr = strconv.FormatFloat(val.Float(), 'f', b10, b64)

	// TODO: dump to text later
	case protoreflect.BytesKind:
		retStr = string(val.Bytes())

	case protoreflect.EnumKind:
		retStr = strconv.Itoa(int(val.Enum()))

	default:
		return "", fmt.Errorf("%v has unknown kind: %v", fd.FullName(), kind)
	}
	return
}
