// Package http http multipart/form-data post 请求数据编解码
package http

import (
	"errors"
	"net/url"

	"git.code.oa.com/trpc-go/trpc-go/codec"
)

var (
	// tagJSON 和json使用相同的tag
	tagJSON = "json"
	// FormDataMarshalType 响应数据序列化方式，默认 json 序列化
	FormDataMarshalType = codec.SerializationTypeJSON
)

func init() {
	codec.RegisterSerializer(
		codec.SerializationTypeFormData,
		NewFormDataSerialization(tagJSON),
	)
}

// getFormDataContentType 获取 form-data 解码的响应头，默认响应 application/json 类型
func getFormDataContentType() string {
	return serializationTypeContentType[FormDataMarshalType]
}

// NewFormDataSerialization 初始化 from 序列化对象
func NewFormDataSerialization(tag string) codec.Serializer {
	return &FormDataSerialization{
		tagName: tag,
	}
}

// FormDataSerialization 打包 http 请求的 kv 结构
type FormDataSerialization struct {
	tagName string
}

// Unmarshal 解包 kv 结构
func (j *FormDataSerialization) Unmarshal(in []byte, body interface{}) error {
	values, err := url.ParseQuery(string(in))
	if err != nil {
		return err
	}
	return unmarshalValues(j.tagName, values, body)
}

// Marshal 序列化
func (j *FormDataSerialization) Marshal(body interface{}) ([]byte, error) {
	serializer := codec.GetSerializer(FormDataMarshalType)
	if serializer == nil {
		return nil, errors.New("empty json serializer")
	}
	return serializer.Marshal(body)
}
