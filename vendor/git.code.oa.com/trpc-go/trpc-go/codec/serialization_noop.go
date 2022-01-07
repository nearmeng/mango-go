package codec

import (
	"errors"
	"fmt"
)

func init() {
	RegisterSerializer(SerializationTypeNoop, &NoopSerialization{})
}

// compile
// 用于编译检查 Body 实现了 BytesBodyIn 方法
var _ BytesBodyIn = &Body{}

// 用于编译检查 Body 实现了 BytesBodyOut 方法
var _ BytesBodyOut = &Body{}

// BytesBodyOut 用于接收自定义类型的 Body
type BytesBodyOut interface {
	Bytes() ([]byte, error)
}

// BytesBody 用于接收自定义类型的 Body
type BytesBodyIn interface {
	SetBytes([]byte) error
}

// Body 二进制body包裹层 不序列化，一般用于网关透传服务
type Body struct {
	Data []byte
}

// String 二进制数据
func (b *Body) String() string {
	return fmt.Sprintf("%v", b.Data)
}

// Write 实现 ByteBodyIn 接口
func (b *Body) SetBytes(p []byte) error {
	if b == nil {
		return errors.New("body nil")
	}
	b.Data = p
	return nil
}

// Bytes 实现 ByteBodyOut 接口
func (b *Body) Bytes() ([]byte, error) {
	if b == nil {
		return nil, errors.New("body nil")
	}
	return b.Data, nil
}

// NoopSerialization 空序列化 用于二进制数据序列化
type NoopSerialization struct {
}

// Unmarshal empty反序列, 因为反序列需要内部解析数据填充到interface，所以对于二进制数据需要通用body包装一层
func (s *NoopSerialization) Unmarshal(in []byte, body interface{}) error {
	bytesBodyIn, ok := body.(BytesBodyIn)
	if ok {
		return bytesBodyIn.SetBytes(in)
	}
	noop, ok := body.(*Body)
	if !ok {
		return errors.New("body type invalid")
	}
	if noop == nil {
		return errors.New("body nil")
	}
	noop.Data = in
	return nil
}

// Marshal empty序列化
func (s *NoopSerialization) Marshal(body interface{}) ([]byte, error) {
	bytesBody, ok := body.(BytesBodyOut)
	if ok {
		return bytesBody.Bytes()
	}
	noop, ok := body.(*Body)
	if !ok {
		return nil, errors.New("body type invalid")
	}
	if noop == nil {
		return nil, errors.New("body nil")
	}
	return noop.Data, nil
}
