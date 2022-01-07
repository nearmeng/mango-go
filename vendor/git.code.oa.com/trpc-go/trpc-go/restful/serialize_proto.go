package restful

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterSerializer(&ProtoSerializer{})
}

var (
	errNotProtoMessageType = errors.New("type is not proto.Message")
)

// ProtoSerializer 用于支持 content-Type: application/octet-stream
type ProtoSerializer struct{}

// Marshal 实现 Serializer
func (*ProtoSerializer) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, errNotProtoMessageType
	}
	return proto.Marshal(msg)
}

// Unmarshal 实现 Serializer
func (*ProtoSerializer) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return errNotProtoMessageType
	}
	return proto.Unmarshal(data, msg)
}

// Name 实现 Serializer
func (*ProtoSerializer) Name() string {
	return "application/octet-stream"
}

// ContentType 实现 Serializer
func (*ProtoSerializer) ContentType() string {
	return "application/octet-stream"
}
