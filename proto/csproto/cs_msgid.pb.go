// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.5.1
// source: cs_msgid.proto

package csproto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type CSMessageID int32

const (
	CSMessageID_cs_message_begin CSMessageID = 0
	CSMessageID_cs_login         CSMessageID = 1
	CSMessageID_cs_message_end   CSMessageID = 4095
)

// Enum value maps for CSMessageID.
var (
	CSMessageID_name = map[int32]string{
		0:    "cs_message_begin",
		1:    "cs_login",
		4095: "cs_message_end",
	}
	CSMessageID_value = map[string]int32{
		"cs_message_begin": 0,
		"cs_login":         1,
		"cs_message_end":   4095,
	}
)

func (x CSMessageID) Enum() *CSMessageID {
	p := new(CSMessageID)
	*p = x
	return p
}

func (x CSMessageID) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CSMessageID) Descriptor() protoreflect.EnumDescriptor {
	return file_cs_msgid_proto_enumTypes[0].Descriptor()
}

func (CSMessageID) Type() protoreflect.EnumType {
	return &file_cs_msgid_proto_enumTypes[0]
}

func (x CSMessageID) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use CSMessageID.Descriptor instead.
func (CSMessageID) EnumDescriptor() ([]byte, []int) {
	return file_cs_msgid_proto_rawDescGZIP(), []int{0}
}

type SCMessageID int32

const (
	SCMessageID_sc_message_begin SCMessageID = 0
	SCMessageID_sc_login         SCMessageID = 1
	SCMessageID_sc_message_end   SCMessageID = 4095
)

// Enum value maps for SCMessageID.
var (
	SCMessageID_name = map[int32]string{
		0:    "sc_message_begin",
		1:    "sc_login",
		4095: "sc_message_end",
	}
	SCMessageID_value = map[string]int32{
		"sc_message_begin": 0,
		"sc_login":         1,
		"sc_message_end":   4095,
	}
)

func (x SCMessageID) Enum() *SCMessageID {
	p := new(SCMessageID)
	*p = x
	return p
}

func (x SCMessageID) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (SCMessageID) Descriptor() protoreflect.EnumDescriptor {
	return file_cs_msgid_proto_enumTypes[1].Descriptor()
}

func (SCMessageID) Type() protoreflect.EnumType {
	return &file_cs_msgid_proto_enumTypes[1]
}

func (x SCMessageID) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use SCMessageID.Descriptor instead.
func (SCMessageID) EnumDescriptor() ([]byte, []int) {
	return file_cs_msgid_proto_rawDescGZIP(), []int{1}
}

var File_cs_msgid_proto protoreflect.FileDescriptor

var file_cs_msgid_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x63, 0x73, 0x5f, 0x6d, 0x73, 0x67, 0x69, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2a, 0x46, 0x0a, 0x0b, 0x43, 0x53, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x49, 0x44, 0x12, 0x14, 0x0a, 0x10, 0x63, 0x73, 0x5f, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x5f, 0x62, 0x65, 0x67, 0x69, 0x6e, 0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08,
	0x63, 0x73, 0x5f, 0x6c, 0x6f, 0x67, 0x69, 0x6e, 0x10, 0x01, 0x12, 0x13, 0x0a, 0x0e, 0x63, 0x73,
	0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x65, 0x6e, 0x64, 0x10, 0xff, 0x1f, 0x2a,
	0x46, 0x0a, 0x0b, 0x53, 0x43, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49, 0x44, 0x12, 0x14,
	0x0a, 0x10, 0x73, 0x63, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x62, 0x65, 0x67,
	0x69, 0x6e, 0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x73, 0x63, 0x5f, 0x6c, 0x6f, 0x67, 0x69, 0x6e,
	0x10, 0x01, 0x12, 0x13, 0x0a, 0x0e, 0x73, 0x63, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x5f, 0x65, 0x6e, 0x64, 0x10, 0xff, 0x1f, 0x42, 0x0a, 0x5a, 0x08, 0x2f, 0x63, 0x73, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_cs_msgid_proto_rawDescOnce sync.Once
	file_cs_msgid_proto_rawDescData = file_cs_msgid_proto_rawDesc
)

func file_cs_msgid_proto_rawDescGZIP() []byte {
	file_cs_msgid_proto_rawDescOnce.Do(func() {
		file_cs_msgid_proto_rawDescData = protoimpl.X.CompressGZIP(file_cs_msgid_proto_rawDescData)
	})
	return file_cs_msgid_proto_rawDescData
}

var file_cs_msgid_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_cs_msgid_proto_goTypes = []interface{}{
	(CSMessageID)(0), // 0: proto.CSMessageID
	(SCMessageID)(0), // 1: proto.SCMessageID
}
var file_cs_msgid_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_cs_msgid_proto_init() }
func file_cs_msgid_proto_init() {
	if File_cs_msgid_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_cs_msgid_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cs_msgid_proto_goTypes,
		DependencyIndexes: file_cs_msgid_proto_depIdxs,
		EnumInfos:         file_cs_msgid_proto_enumTypes,
	}.Build()
	File_cs_msgid_proto = out.File
	file_cs_msgid_proto_rawDesc = nil
	file_cs_msgid_proto_goTypes = nil
	file_cs_msgid_proto_depIdxs = nil
}
