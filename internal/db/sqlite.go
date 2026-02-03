package db

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/rowjay/db-backup-utility/internal/config"
	"github.com/rowjay/db-backup-utility/internal/storage"
)

type SQLiteAdapter struct{}

func NewSQLiteAdapter() *SQLiteAdapter { return &SQLiteAdapter{} }

func (s *SQLiteAdapter) Name() string { return "sqlite" }

func (s *SQLiteAdapter) Capabilities() Capabilities {
	return Capabilities{Incremental: false, Differential: false}
}

func (s *SQLiteAdapter) Validate(ctx context.Context, cfg config.DatabaseConfig) error {
	if cfg.SQLitePath == "" {
		return fmt.Errorf("sqlite_path is required")
	}
	if _, err := os.Stat(cfg.SQLitePath); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteAdapter) Dump(ctx context.Context, cfg config.DatabaseConfig, backup config.BackupConfig) (*DumpStream, error) {
	if cfg.SQLitePath == "" {
		return nil, fmt.Errorf("sqlite_path is required")
	}
	file, err := os.Open(cfg.SQLitePath)
	if err != nil {
		return nil, err
	}
	return &DumpStream{Reader: file, Wait: file.Close}, nil
}

func (s *SQLiteAdapter) Restore(ctx context.Context, cfg config.DatabaseConfig, restore config.RestoreConfig, manifest storage.Manifest) (*RestoreStream, error) {
	if cfg.SQLitePath == "" {
		return nil, fmt.Errorf("sqlite_path is required")
	}
	if !restore.DropExisting {
		if _, err := os.Stat(cfg.SQLitePath); err == nil {
			return nil, fmt.Errorf("sqlite file already exists; enable drop_existing to overwrite")
		}
	}
	file, err := os.OpenFile(cfg.SQLitePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	writer := &flushWriter{writer: file}
	return &RestoreStream{Writer: writer, Wait: writer.Close}, nil
}

type flushWriter struct {
	writer *os.File
}

func (f *flushWriter) Write(p []byte) (int, error) { return f.writer.Write(p) }

func (f *flushWriter) Close() error {
	if err := f.writer.Sync(); err != nil {
		return err
	}
	return f.writer.Close()
}

var _ io.WriteCloser = (*flushWriter)(nil)
