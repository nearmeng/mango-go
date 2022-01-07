package codec

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"sync"
)

func init() {
	RegisterCompressor(CompressTypeGzip, &GzipCompress{})
}

// GzipCompress gzip解压缩
type GzipCompress struct {
	readerPool sync.Pool
	writerPool sync.Pool
}

// Compress gzip压缩
func (c *GzipCompress) Compress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}

	buffer := &bytes.Buffer{}
	z, ok := c.writerPool.Get().(*gzip.Writer)
	if !ok {
		z = gzip.NewWriter(buffer)
	} else {
		z.Reset(buffer)
	}
	defer c.writerPool.Put(z)

	if _, err := z.Write(in); err != nil {
		return nil, err
	}
	if err := z.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// Decompress gzip解压
func (c *GzipCompress) Decompress(in []byte) ([]byte, error) {
	if len(in) == 0 {
		return in, nil
	}
	br := bytes.NewReader(in)
	z, ok := c.readerPool.Get().(*gzip.Reader)
	defer func() {
		if z != nil {
			c.readerPool.Put(z)
		}
	}()
	if !ok {
		gr, err := gzip.NewReader(br)
		if err != nil {
			return nil, err
		}
		z = gr
	} else {
		if err := z.Reset(br); err != nil {
			return nil, err
		}
	}
	out, err := ioutil.ReadAll(z)
	if err != nil {
		return nil, err
	}
	return out, nil
}
