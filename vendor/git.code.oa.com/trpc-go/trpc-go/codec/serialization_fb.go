package codec

import (
	"errors"

	flatbuffers "github.com/google/flatbuffers/go"
)

func init() {
	RegisterSerializer(SerializationTypeFlatBuffer, &FBSerialization{})
}

// FBSerialization 提供 flatbuffers 的序列化方式
// flatbuffers 官网见 https://google.github.io/flatbuffers
type FBSerialization struct{}

// Unmarshal 对字节流进行反序列化操作
func (*FBSerialization) Unmarshal(in []byte, body interface{}) error {
	body, ok := body.(flatbuffersInit)
	if !ok {
		return errors.New("unmarshal fail: body does not implement flatbufferInit interface")
	}
	body.(flatbuffersInit).Init(in, flatbuffers.GetUOffsetT(in))
	return nil
}

// Marshal 对 flatbuffers 进行序列化操作
func (*FBSerialization) Marshal(body interface{}) ([]byte, error) {
	builder, ok := body.(*flatbuffers.Builder)
	if !ok {
		return nil, errors.New("marshal fail: body not *flatbuffers.Builder")
	}
	return builder.FinishedBytes(), nil
}

type flatbuffersInit interface {
	Init(data []byte, i flatbuffers.UOffsetT)
}
