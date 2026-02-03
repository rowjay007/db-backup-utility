package storage

import (
	"context"
	"io"
	"time"
)

type ObjectInfo struct {
	Key        string
	Size       int64
	Modified   time.Time
	ETag       string
	Metadata   map[string]string
	IsManifest bool
}

type Storage interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64, metadata map[string]string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
