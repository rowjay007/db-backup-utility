package compress

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

const (
	TypeNone = "none"
	TypeGzip = "gzip"
	TypeZstd = "zstd"
)

func WrapWriter(kind string, w io.Writer) (io.WriteCloser, error) {
	switch kind {
	case "", TypeNone:
		return nopWriteCloser{w}, nil
	case TypeGzip:
		return gzip.NewWriter(w), nil
	case TypeZstd:
		return zstd.NewWriter(w)
	default:
		return nil, fmt.Errorf("unsupported compression: %s", kind)
	}
}

func WrapReader(kind string, r io.Reader) (io.ReadCloser, error) {
	switch kind {
	case "", TypeNone:
		return io.NopCloser(r), nil
	case TypeGzip:
		return gzip.NewReader(r)
	case TypeZstd:
		dec, err := zstd.NewReader(r)
		if err != nil {
			return nil, err
		}
		return zstdReadCloser{Decoder: dec}, nil
	default:
		return nil, fmt.Errorf("unsupported compression: %s", kind)
	}
}

type nopWriteCloser struct{ io.Writer }

func (n nopWriteCloser) Close() error { return nil }

type zstdReadCloser struct{ *zstd.Decoder }

func (z zstdReadCloser) Close() error {
	z.Decoder.Close()
	return nil
}
