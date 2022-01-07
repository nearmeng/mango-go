package restful

import (
	"errors"
	"net/url"
	"strings"

	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterSerializer(&FormSerializer{})
}

// FormSerializer 用于支持 Content-Type: application/x-www-form-urlencoded
type FormSerializer struct {
	// If DiscardUnknown is set, unknown fields are ignored.
	DiscardUnknown bool
}

// Marshal 实现 Serializer
// form 的 marshal 和 jsonpb 的 marshal 是一致的
func (*FormSerializer) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok { // marshal tRPC message 的其中一个字段
		return marshal(v)
	}
	// marshal tRPC message
	return Marshaller.Marshal(msg)
}

// Unmarshal 实现 Serializer
func (f *FormSerializer) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return errNotProtoMessageType
	}

	// 获取 url.Values
	vs, err := url.ParseQuery(string(data))
	if err != nil {
		return err
	}
	// 填充 proto message
	for key, values := range vs {
		fieldPath := strings.Split(key, ".")
		if err := PopulateMessage(msg, fieldPath, values); err != nil {
			if !f.DiscardUnknown || !errors.Is(err, ErrTraverseNotFound) {
				return err
			}
		}
	}

	return nil
}

// Name 实现 Serializer
func (*FormSerializer) Name() string {
	return "application/x-www-form-urlencoded"
}

// ContentType 实现 Serializer
// 和 jsonpb 一致
func (*FormSerializer) ContentType() string {
	return "application/json"
}
