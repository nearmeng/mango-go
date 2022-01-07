package codec

import (
	"bufio"
	"io"
)

// DefaultReaderSize 定义Reader读缓冲区默认值 (单位B)
const DefaultReaderSize = 4 * 1024

// readerSizeConfig framer读包时框架配置的缓冲区大小
var readerSizeConfig = DefaultReaderSize

// NewReaderSize 封装Reader，为Reader提供读缓冲区
// size <= 0: 不提供读缓冲区功能
func NewReaderSize(r io.Reader, size int) io.Reader {
	if size <= 0 {
		return r
	}
	return bufio.NewReaderSize(r, size)
}

// NewReader 封装Reader, 使用系统配置的缓冲区大小
func NewReader(r io.Reader) io.Reader {
	return bufio.NewReaderSize(r, readerSizeConfig)
}

// GetReaderSize 获取网络读缓冲区大小(单位B)
func GetReaderSize() int {
	return readerSizeConfig
}

// SetReaderSize 设置网络读缓冲区大小(单位B)
func SetReaderSize(size int) {
	readerSizeConfig = size
}

// FramerBuilder 通常每个连接Build一个Framer
type FramerBuilder interface {
	New(io.Reader) Framer
}

// Framer 读写数据桢
type Framer interface {
	ReadFrame() ([]byte, error)
}

// SafeFramer 提供方法描述Framer读取的数据是否并发安全
type SafeFramer interface {
	Framer
	// 判断Framer是否支持并发读包
	IsSafe() bool
}

// IsSafeFramer 判断Framer是否为SafeFramer
func IsSafeFramer(f interface{}) bool {
	framer, ok := f.(SafeFramer)
	if ok && framer.IsSafe() {
		return true
	}
	return false
}

// Decoder 解码回包
type Decoder interface {
	// Decode 解析出帧头，包头，包体
	Decode() (TransportResponseFrame, error)
	// UpdateMsg 更新 Msg，Decode 解析出的回包作为参数传入
	UpdateMsg(interface{}, Msg) error
}

// TransportResponseFrame Decode 解析出的回包应该实现接口
type TransportResponseFrame interface {
	// GetRequestId 返回本次请求的 RequestId
	GetRequestID() uint32
	// GetResponseBuf 返回本次请求的包体
	GetResponseBuf() []byte
}
