package restful

import (
	"io"
	"net/http"
)

// Compressor 用于 http body 压缩/解压缩
type Compressor interface {
	// Compress 压缩
	Compress(w io.Writer) (io.WriteCloser, error)
	// Decompress 解压缩
	Decompress(r io.Reader) (io.Reader, error)
	// Name 表示 Compressor 名字
	Name() string
	// ContentEncoding 表示 http 回包时设置的 Content-Encoding
	ContentEncoding() string
}

// Compressor 相关 http header
var (
	headerAcceptEncoding  = http.CanonicalHeaderKey("Accept-Encoding")
	headerContentEncoding = http.CanonicalHeaderKey("Content-Encoding")
)

var compressors = make(map[string]Compressor)

// RegisterCompressor 注册 Compressor，非线程安全，只能在 init 函数中使用
func RegisterCompressor(c Compressor) {
	if c == nil || c.Name() == "" {
		panic("tried to register nil or anonymous compressor")
	}
	compressors[c.Name()] = c
}

// GetCompressor 获取 Compressor
func GetCompressor(name string) Compressor {
	return compressors[name]
}

// compressorForTranscoding 获取 Compressor 用于转码
func compressorForTranscoding(contentEncodings []string, acceptEncodings []string) (Compressor, Compressor) {
	var reqCompressor, respCompressor Compressor // 都可以为 nil

	for _, contentEncoding := range contentEncodings {
		if c, ok := compressors[contentEncoding]; ok {
			reqCompressor = c
			break
		}
	}

	for _, acceptEncoding := range acceptEncodings {
		if c, ok := compressors[acceptEncoding]; ok {
			respCompressor = c
			break
		}
	}

	return reqCompressor, respCompressor
}
