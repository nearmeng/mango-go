package attaapi

import (
	"bytes"
	"sync"
)

var bufferPool sync.Pool

func init() {
	bufferPool.New = func() interface{} {
		return &bytes.Buffer{}
	}
}

//获取缓存
func GetBuf() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

//归还缓存
func PutBuf(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}
