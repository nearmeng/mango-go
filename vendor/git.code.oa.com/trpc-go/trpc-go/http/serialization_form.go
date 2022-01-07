package http

import (
	"net/url"

	"git.code.oa.com/trpc-go/trpc-go/codec"

	"github.com/go-playground/form"
	"github.com/mitchellh/mapstructure"
)

// 和json使用相同的tag
var tag = "json"

func init() {
	codec.RegisterSerializer(
		codec.SerializationTypeForm,
		NewFormSerialization(tag),
	)
}

// NewFormSerialization 初始化from序列化对象
func NewFormSerialization(tag string) codec.Serializer {
	encoder := form.NewEncoder()
	encoder.SetTagName(tag)
	return &FormSerialization{
		tagname: tag,
		encoder: encoder,
	}
}

// FormSerialization 打包http get请求的kv结构
type FormSerialization struct {
	tagname string
	encoder *form.Encoder
}

// Unmarshal 解包kv结构
func (j *FormSerialization) Unmarshal(in []byte, body interface{}) error {
	values, err := url.ParseQuery(string(in))
	if err != nil {
		return err
	}
	return unmarshalValues(j.tagname, values, body)
}

// unmarshalValues 根据 tagname 解析 values 中对应的字段
func unmarshalValues(tagname string, values url.Values, body interface{}) error {
	params := map[string]interface{}{}
	for k, v := range values {
		if len(v) == 1 {
			params[k] = v[0]
		} else {
			params[k] = v
		}
	}
	config := &mapstructure.DecoderConfig{TagName: tagname, Result: body, WeaklyTypedInput: true, Metadata: nil}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(params)
}

// Marshal 打包kv结构
func (j *FormSerialization) Marshal(body interface{}) ([]byte, error) {
	if req, ok := body.(url.Values); ok { // 用于向后端post发送form urlencode请求
		return []byte(req.Encode()), nil
	}
	val, err := j.encoder.Encode(body)
	if err != nil {
		return nil, err
	}
	return []byte(val.Encode()), nil
}
