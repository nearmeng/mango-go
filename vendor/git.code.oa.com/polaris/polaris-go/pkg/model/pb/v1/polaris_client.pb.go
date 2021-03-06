// Code generated by protoc-gen-go.
// source: polaris_client.proto
// DO NOT EDIT!

/*
Package v1 is a generated protocol buffer package.

It is generated from these files:
	polaris_client.proto
	polaris_grpcapi.proto
	polaris_model.proto
	polaris_ratelimitrule.proto
	polaris_request.proto
	polaris_response.proto
	polaris_routing.proto
	polaris_service.proto

It has these top-level messages:
	Client
	Location
	MatchString
	RateLimit
	Rule
	Amount
	Report
	AmountAdjuster
	ClimbConfig
	DiscoverRequest
	SimpleResponse
	Response
	BatchWriteResponse
	BatchQueryResponse
	DiscoverResponse
	Routing
	Route
	Source
	Destination
	Namespace
	Service
	ServiceAlias
	Instance
	HealthCheck
	HeartbeatHealthCheck
*/
package v1

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/wrappers"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Client_ClientType int32

const (
	Client_UNKNOWN Client_ClientType = 0
	Client_SDK     Client_ClientType = 1
	Client_AGENT   Client_ClientType = 2
)

var Client_ClientType_name = map[int32]string{
	0: "UNKNOWN",
	1: "SDK",
	2: "AGENT",
}
var Client_ClientType_value = map[string]int32{
	"UNKNOWN": 0,
	"SDK":     1,
	"AGENT":   2,
}

func (x Client_ClientType) String() string {
	return proto.EnumName(Client_ClientType_name, int32(x))
}
func (Client_ClientType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type Client struct {
	Host     *google_protobuf.StringValue `protobuf:"bytes,1,opt,name=host" json:"host,omitempty"`
	Type     Client_ClientType            `protobuf:"varint,2,opt,name=type,enum=v1.Client_ClientType" json:"type,omitempty"`
	Version  *google_protobuf.StringValue `protobuf:"bytes,3,opt,name=version" json:"version,omitempty"`
	Location *Location                    `protobuf:"bytes,4,opt,name=location" json:"location,omitempty"`
}

func (m *Client) Reset()                    { *m = Client{} }
func (m *Client) String() string            { return proto.CompactTextString(m) }
func (*Client) ProtoMessage()               {}
func (*Client) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Client) GetHost() *google_protobuf.StringValue {
	if m != nil {
		return m.Host
	}
	return nil
}

func (m *Client) GetType() Client_ClientType {
	if m != nil {
		return m.Type
	}
	return Client_UNKNOWN
}

func (m *Client) GetVersion() *google_protobuf.StringValue {
	if m != nil {
		return m.Version
	}
	return nil
}

func (m *Client) GetLocation() *Location {
	if m != nil {
		return m.Location
	}
	return nil
}

func init() {
	proto.RegisterType((*Client)(nil), "v1.Client")
	proto.RegisterEnum("v1.Client_ClientType", Client_ClientType_name, Client_ClientType_value)
}

func init() { proto.RegisterFile("polaris_client.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 247 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x29, 0xc8, 0xcf, 0x49,
	0x2c, 0xca, 0x2c, 0x8e, 0x4f, 0xce, 0xc9, 0x4c, 0xcd, 0x2b, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9,
	0x17, 0x62, 0x2a, 0x33, 0x94, 0x92, 0x4b, 0xcf, 0xcf, 0x4f, 0xcf, 0x49, 0xd5, 0x07, 0x8b, 0x24,
	0x95, 0xa6, 0xe9, 0x97, 0x17, 0x25, 0x16, 0x14, 0xa4, 0x16, 0x15, 0x43, 0xd4, 0x48, 0x09, 0xc3,
	0x74, 0xe6, 0xe6, 0xa7, 0xa4, 0xe6, 0x40, 0x04, 0x95, 0xbe, 0x31, 0x72, 0xb1, 0x39, 0x83, 0x4d,
	0x12, 0x32, 0xe0, 0x62, 0xc9, 0xc8, 0x2f, 0x2e, 0x91, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x36, 0x92,
	0xd1, 0x83, 0x18, 0xa7, 0x07, 0x33, 0x4e, 0x2f, 0xb8, 0xa4, 0x28, 0x33, 0x2f, 0x3d, 0x2c, 0x31,
	0xa7, 0x34, 0x35, 0x08, 0xac, 0x52, 0x48, 0x93, 0x8b, 0xa5, 0xa4, 0xb2, 0x20, 0x55, 0x82, 0x49,
	0x81, 0x51, 0x83, 0xcf, 0x48, 0x54, 0xaf, 0xcc, 0x50, 0x0f, 0x62, 0x16, 0x94, 0x0a, 0xa9, 0x2c,
	0x48, 0x0d, 0x02, 0x2b, 0x11, 0x32, 0xe3, 0x62, 0x2f, 0x4b, 0x2d, 0x2a, 0xce, 0xcc, 0xcf, 0x93,
	0x60, 0x26, 0xc2, 0x7c, 0x98, 0x62, 0x21, 0x0d, 0x2e, 0x8e, 0x9c, 0xfc, 0xe4, 0xc4, 0x12, 0x90,
	0x46, 0x16, 0xb0, 0x46, 0x1e, 0x90, 0x35, 0x3e, 0x50, 0xb1, 0x20, 0xb8, 0xac, 0x92, 0x2e, 0x17,
	0x17, 0xc2, 0x56, 0x21, 0x6e, 0x2e, 0xf6, 0x50, 0x3f, 0x6f, 0x3f, 0xff, 0x70, 0x3f, 0x01, 0x06,
	0x21, 0x76, 0x2e, 0xe6, 0x60, 0x17, 0x6f, 0x01, 0x46, 0x21, 0x4e, 0x2e, 0x56, 0x47, 0x77, 0x57,
	0xbf, 0x10, 0x01, 0xa6, 0x24, 0x36, 0xb0, 0xbd, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0x29,
	0x22, 0x90, 0x02, 0x50, 0x01, 0x00, 0x00,
}
