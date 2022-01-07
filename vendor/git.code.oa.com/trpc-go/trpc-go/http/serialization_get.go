package http

import (
	"errors"
	"net/url"

	"git.code.oa.com/trpc-go/trpc-go/codec"
)

func init() {
	codec.RegisterSerializer(codec.SerializationTypeGet, NewGetSerialization(tag))
}

// NewGetSerialization 初始化Get序列化对象
func NewGetSerialization(tag string) codec.Serializer {

	return &GetSerialization{
		tagname: tag,
	}
}

// GetSerialization 打包http get请求的kv结构
type GetSerialization struct {
	tagname string
}

// Unmarshal 解包kv结构
func (s *GetSerialization) Unmarshal(in []byte, body interface{}) error {
	values, err := url.ParseQuery(string(in))
	if err != nil {
		return err
	}
	return unmarshalValues(s.tagname, values, body)
}

// Marshal 打包kv结构
func (s *GetSerialization) Marshal(body interface{}) ([]byte, error) {
	jsonSerializer := codec.GetSerializer(codec.SerializationTypeJSON) // 用于收到Get请求给前端回json包
	if jsonSerializer == nil {
		return nil, errors.New("empty json serializer")
	}
	return jsonSerializer.Marshal(body)
}
