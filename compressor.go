package go_xml

import (
	"bytes"
	"compress/gzip"
	"sync"
)

type Compressor interface {
	Compress(data *bytes.Buffer) (*bytes.Buffer, error)
}

type GzipCompressor struct{}

func (gc *GzipCompressor) Compress(data *bytes.Buffer) (*bytes.Buffer, error) {
	compressedBuffer := acquireBuffer()
	writer := gzip.NewWriter(compressedBuffer)

	if _, err := writer.Write(data.Bytes()); err != nil {
		releaseBuffer(compressedBuffer)
		return nil, err
	}
	writer.Close()

	return compressedBuffer, nil
}

var compressorPool = sync.Pool{
	New: func() interface{} {
		return &GzipCompressor{}
	},
}

func acquireCompressor() Compressor {
	return compressorPool.Get().(Compressor)
}

func releaseCompressor(compressor Compressor) {
	compressorPool.Put(compressor)
}
