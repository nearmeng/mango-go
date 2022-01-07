package restful

import (
	"net/http"
)

// Serializer 用于 http body 和 tRPC message 或其中一个字段的序列化/反序列化
type Serializer interface {
	// Marshal 把 tRPC message 或其中一个字段序列化到 http body
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal 把 http body 反序列化到 tRPC message 或其中一个字段
	Unmarshal(data []byte, v interface{}) error
	// Name Serializer 名字
	Name() string
	// ContentType http 回包时设置的 Content-Type
	ContentType() string
}

// 默认为 jsonpb
var defaultSerializer Serializer = &JSONPBSerializer{}

// Serializer 相关 http header
var (
	headerAccept      = http.CanonicalHeaderKey("Accept")
	headerContentType = http.CanonicalHeaderKey("Content-Type")
)

var serializers = make(map[string]Serializer)

// RegisterSerializer 注册 Serializer, 非线程安全，只允许在 init 函数中使用
func RegisterSerializer(s Serializer) {
	if s == nil || s.Name() == "" {
		panic("tried to register nil or anonymous serializer")
	}
	serializers[s.Name()] = s
}

// SetDefaultSerializer 设置默认 Serializer, 非线程安全，只允许在 init 函数中使用
func SetDefaultSerializer(s Serializer) {
	if s == nil || s.Name() == "" {
		panic("tried to set nil or anonymous serializer as the default serializer")
	}
	defaultSerializer = s
}

// GetSerializer 获取 Serializer
func GetSerializer(name string) Serializer {
	return serializers[name]
}

// serializerForTranscoding 获取 Serializer 用于转码
func serializerForTranscoding(contentTypes []string, accepts []string) (Serializer, Serializer) {
	var reqSerializer, respSerializer Serializer // 收包和回包 Serializer 都不允许为 nil

	// ContentType => Req Serializer
	for _, contentType := range contentTypes {
		if s, ok := serializers[contentType]; ok {
			reqSerializer = s
			break
		}
	}

	// Accept => Resp Serializer
	for _, accept := range accepts {
		if s, ok := serializers[accept]; ok {
			respSerializer = s
			break
		}
	}

	if reqSerializer == nil { // 收包 Serializer 获取不到就使用 defaultSerializer
		reqSerializer = defaultSerializer
	}
	if respSerializer == nil { // 回包 Serializer 获取不到就使用收包 Serializer
		respSerializer = reqSerializer
	}

	return reqSerializer, respSerializer
}
