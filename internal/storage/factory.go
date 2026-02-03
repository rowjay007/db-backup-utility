package storage

import (
	"fmt"

	"github.com/rowjay/db-backup-utility/internal/config"
)

func New(cfg config.StorageConfig) (Storage, error) {
	switch cfg.Backend {
	case "local", "":
		return NewLocal(cfg.Local.Path), nil
	case "s3":
		if cfg.S3.Endpoint == "" || cfg.S3.Bucket == "" {
			return nil, fmt.Errorf("s3 endpoint and bucket are required")
		}
		return NewS3(cfg.S3.Endpoint, cfg.S3.Region, cfg.S3.Bucket, cfg.S3.AccessKey, cfg.S3.SecretKey, cfg.S3.SessionToken, cfg.S3.UseSSL, cfg.S3.ForcePathStyle, cfg.S3.TLSInsecureSkip)
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", cfg.Backend)
	}
}
