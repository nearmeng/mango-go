package restful

import (
	"compress/gzip"
	"io"
	"sync"
)

func init() {
	RegisterCompressor(&GZIPCompressor{})
}

var readerPool sync.Pool
var writerPool sync.Pool

// GZIPCompressor 用于支持 Content-Encoding: gzip
type GZIPCompressor struct{}

// 包一层，实现池化
type wrappedWriter struct {
	*gzip.Writer
}

// Close 重写 Close 方法，Close 后放回内存池内
func (w *wrappedWriter) Close() error {
	defer writerPool.Put(w)
	return w.Writer.Close()
}

// 包一层，实现池化
type wrappedReader struct {
	*gzip.Reader
}

// Read 重写 Read 方法，Read 完后放回内存池内
func (r *wrappedReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF {
		readerPool.Put(r)
	}
	return n, err
}

// Compress 实现 Compressor
func (*GZIPCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	z, ok := writerPool.Get().(*wrappedWriter)
	if !ok {
		z = &wrappedWriter{
			Writer: gzip.NewWriter(w),
		}
	}
	z.Writer.Reset(w)
	return z, nil
}

// Decompress 实现 Compressor
func (g *GZIPCompressor) Decompress(r io.Reader) (io.Reader, error) {
	z, ok := readerPool.Get().(*wrappedReader)
	if !ok {
		gzipReader, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		return &wrappedReader{
			Reader: gzipReader,
		}, nil
	}
	if err := z.Reader.Reset(r); err != nil {
		readerPool.Put(z)
		return nil, err
	}
	return z, nil
}

// Name 实现 Compressor
func (*GZIPCompressor) Name() string {
	return "gzip"
}

// ContentEncoding 实现 Compressor
func (*GZIPCompressor) ContentEncoding() string {
	return "gzip"
}
