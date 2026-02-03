package db

import (
	"context"
	"fmt"
	"io"

	"github.com/rowjay/db-backup-utility/internal/config"
	"github.com/rowjay/db-backup-utility/internal/storage"
)

type Adapter interface {
	Name() string
	Validate(ctx context.Context, cfg config.DatabaseConfig) error
	Dump(ctx context.Context, cfg config.DatabaseConfig, backup config.BackupConfig) (*DumpStream, error)
	Restore(ctx context.Context, cfg config.DatabaseConfig, restore config.RestoreConfig, manifest storage.Manifest) (*RestoreStream, error)
	Capabilities() Capabilities
}

type Capabilities struct {
	Incremental       bool
	Differential      bool
	TableRestore      bool
	CollectionRestore bool
}

type DumpStream struct {
	Reader io.ReadCloser
	Wait   func() error
}

type RestoreStream struct {
	Writer io.WriteCloser
	Wait   func() error
}

func NewAdapter(dbType string, allowMissingTools bool) (Adapter, error) {
	switch dbType {
	case "postgres", "postgresql":
		return NewPostgresAdapter(allowMissingTools), nil
	case "mysql", "mariadb":
		return NewMySQLAdapter(allowMissingTools), nil
	case "mongodb", "mongo":
		return NewMongoAdapter(allowMissingTools), nil
	case "sqlite", "sqlite3":
		return NewSQLiteAdapter(), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
