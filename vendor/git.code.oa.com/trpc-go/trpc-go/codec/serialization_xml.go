package codec

import (
	"encoding/xml"
)

func init() {
	RegisterSerializer(SerializationTypeXML, &XMLSerialization{})
}

// XMLSerialization 序列化 xml 包体
type XMLSerialization struct{}

// Unmarshal 反序列化 xml
func (*XMLSerialization) Unmarshal(in []byte, body interface{}) error {
	return xml.Unmarshal(in, body)
}

// Marshal 序列化 xml
func (*XMLSerialization) Marshal(body interface{}) ([]byte, error) {
	return xml.Marshal(body)
}
