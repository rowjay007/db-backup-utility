package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/rowjay/db-backup-utility/internal/compress"
	"github.com/rowjay/db-backup-utility/internal/config"
	"github.com/rowjay/db-backup-utility/internal/cryptoutil"
	"github.com/rowjay/db-backup-utility/internal/db"
	"github.com/rowjay/db-backup-utility/internal/lock"
	"github.com/rowjay/db-backup-utility/internal/notify"
	"github.com/rowjay/db-backup-utility/internal/storage"
	"github.com/rowjay/db-backup-utility/internal/util"
	"github.com/rowjay/db-backup-utility/internal/version"
)

type App struct {
	Cfg      *config.Config
	Adapter  db.Adapter
	Storage  storage.Storage
	Log      zerolog.Logger
	Notifier notify.Notifier
}

func New(cfg *config.Config, adapter db.Adapter, store storage.Storage, log zerolog.Logger, notifier notify.Notifier) *App {
	return &App{Cfg: cfg, Adapter: adapter, Storage: store, Log: log, Notifier: notifier}
}

type BackupResult struct {
	Manifest storage.Manifest
	Key      string
}

func (a *App) Backup(ctx context.Context) (*BackupResult, error) {
	start := time.Now()
	var opErr error
	var key string
	defer func() {
		if a.Notifier == nil {
			return
		}
		event := notify.Event{
			Type:      "backup",
			Message:   fmt.Sprintf("backup %s", a.Cfg.Database.Database),
			Status:    statusFromErr(opErr),
			Database:  a.Cfg.Database.Database,
			DBType:    a.Cfg.Database.Type,
			StartedAt: start,
			EndedAt:   time.Now(),
			Duration:  time.Since(start).String(),
			Key:       key,
		}
		if opErr != nil {
			event.Error = opErr.Error()
		}
		_ = a.Notifier.Notify(context.Background(), event)
	}()

	guard, err := lock.Acquire(a.Cfg.Global.LockFile)
	if err != nil {
		opErr = err
		return nil, err
	}
	defer guard.Release()

	ok, err := util.InWindow(time.Now(), a.Cfg.Schedule.WindowStart, a.Cfg.Schedule.WindowEnd, a.Cfg.Schedule.Timezone)
	if err != nil {
		return nil, err
	}
	if !ok {
		opErr = fmt.Errorf("current time is outside configured backup window")
		return nil, opErr
	}
	if err := a.Adapter.Validate(ctx, a.Cfg.Database); err != nil {
		opErr = err
		return nil, err
	}
	caps := a.Adapter.Capabilities()
	if strings.EqualFold(a.Cfg.Backup.Type, "incremental") && !caps.Incremental {
		opErr = fmt.Errorf("incremental backups are not supported for %s", a.Adapter.Name())
		return nil, opErr
	}
	if strings.EqualFold(a.Cfg.Backup.Type, "differential") && !caps.Differential {
		opErr = fmt.Errorf("differential backups are not supported for %s", a.Adapter.Name())
		return nil, opErr
	}
	if a.Cfg.Backup.Encryption && a.Cfg.Backup.EncryptionKey == "" {
		opErr = fmt.Errorf("encryption is enabled but encryption_key is empty")
		return nil, opErr
	}

	ext := buildExtension(a.Cfg.Backup.Compression, a.Cfg.Backup.Encryption)
	key = util.BuildObjectKey(a.Cfg.Storage.Prefix, a.Cfg.Database.Type, a.Cfg.Database.Database, a.Cfg.Backup.Type, time.Now(), ext)

	if a.Cfg.Backup.Idempotent {
		exists, err := a.Storage.Exists(ctx, key)
		if err != nil {
			opErr = err
			return nil, err
		}
		if exists {
			opErr = fmt.Errorf("backup already exists: %s", key)
			return nil, opErr
		}
	}

	dumpStream, err := a.Adapter.Dump(ctx, a.Cfg.Database, a.Cfg.Backup)
	if err != nil {
		opErr = err
		return nil, err
	}
	defer dumpStream.Reader.Close()

	pipeReader, pipeWriter := io.Pipe()
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer pipeReader.Close()
		return a.Storage.Put(egCtx, key, pipeReader, -1, map[string]string{"dbu-backup": "true"})
	})

	eg.Go(func() error {
		writer := io.Writer(pipeWriter)
		closers := []io.Closer{pipeWriter}
		if a.Cfg.Backup.Compression != "" && a.Cfg.Backup.Compression != compress.TypeNone {
			compWriter, err := compress.WrapWriter(a.Cfg.Backup.Compression, writer)
			if err != nil {
				_ = pipeWriter.CloseWithError(err)
				return err
			}
			writer = compWriter
			closers = append(closers, compWriter)
		}
		if a.Cfg.Backup.Encryption {
			keyBytes, err := cryptoutil.ParseKey(a.Cfg.Backup.EncryptionKey)
			if err != nil {
				_ = pipeWriter.CloseWithError(err)
				return err
			}
			encWriter, err := cryptoutil.EncryptWriter(writer, keyBytes)
			if err != nil {
				_ = pipeWriter.CloseWithError(err)
				return err
			}
			writer = encWriter
			closers = append(closers, encWriter)
		}
		_, err := io.Copy(writer, dumpStream.Reader)
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			return err
		}
		for i := len(closers) - 1; i >= 0; i-- {
			if err := closers[i].Close(); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return err
			}
		}
		if err := pipeWriter.Close(); err != nil {
			_ = pipeWriter.CloseWithError(err)
			return err
		}
		return nil
	})

	if err := dumpStream.Wait(); err != nil {
		_ = pipeWriter.CloseWithError(err)
		_ = eg.Wait()
		opErr = err
		return nil, err
	}
	if err := eg.Wait(); err != nil {
		opErr = err
		return nil, err
	}

	stat, err := a.Storage.Stat(ctx, key)
	if err != nil {
		opErr = err
		return nil, err
	}
	manifest := storage.Manifest{
		ID:           fmt.Sprintf("%s-%d", a.Cfg.Database.Database, time.Now().UnixNano()),
		Key:          key,
		DatabaseType: a.Cfg.Database.Type,
		Database:     a.Cfg.Database.Database,
		BackupType:   a.Cfg.Backup.Type,
		Compression:  a.Cfg.Backup.Compression,
		Encryption:   a.Cfg.Backup.Encryption,
		CreatedAt:    time.Now().UTC(),
		SizeBytes:    stat.Size,
		Tables:       a.Cfg.Backup.Tables,
		Collections:  a.Cfg.Backup.Collections,
		ToolVersion:  version.Version,
	}

	if err := a.writeManifest(ctx, manifest); err != nil {
		a.Log.Warn().Err(err).Msg("failed to write manifest")
	}

	_ = a.applyRetention(ctx)

	return &BackupResult{Manifest: manifest, Key: key}, nil
}

func (a *App) Restore(ctx context.Context, key string) error {
	start := time.Now()
	var opErr error
	defer func() {
		if a.Notifier == nil {
			return
		}
		event := notify.Event{
			Type:      "restore",
			Message:   fmt.Sprintf("restore %s", a.Cfg.Database.Database),
			Status:    statusFromErr(opErr),
			Database:  a.Cfg.Database.Database,
			DBType:    a.Cfg.Database.Type,
			StartedAt: start,
			EndedAt:   time.Now(),
			Duration:  time.Since(start).String(),
			Key:       key,
		}
		if opErr != nil {
			event.Error = opErr.Error()
		}
		_ = a.Notifier.Notify(context.Background(), event)
	}()

	guard, err := lock.Acquire(a.Cfg.Global.LockFile)
	if err != nil {
		opErr = err
		return err
	}
	defer guard.Release()

	if err := a.Adapter.Validate(ctx, a.Cfg.Database); err != nil {
		opErr = err
		return err
	}
	manifest, _ := a.readManifest(ctx, key)

	if a.Cfg.Restore.DryRun {
		a.Log.Info().Str("key", key).Msg("dry run restore")
		return nil
	}

	reader, err := a.Storage.Get(ctx, key)
	if err != nil {
		opErr = err
		return err
	}
	defer reader.Close()

	payload := io.Reader(reader)
	if manifest.Encryption || a.Cfg.Backup.Encryption {
		if a.Cfg.Backup.EncryptionKey == "" {
			opErr = fmt.Errorf("encryption key is required to restore encrypted backup")
			return opErr
		}
		keyBytes, err := cryptoutil.ParseKey(a.Cfg.Backup.EncryptionKey)
		if err != nil {
			opErr = err
			return err
		}
		payload, err = cryptoutil.DecryptReader(payload, keyBytes)
		if err != nil {
			opErr = err
			return err
		}
	}

	compression := manifest.Compression
	if compression == "" {
		compression = a.Cfg.Backup.Compression
	}
	compReader, err := compress.WrapReader(compression, payload)
	if err != nil {
		opErr = err
		return err
	}
	defer compReader.Close()

	restoreStream, err := a.Adapter.Restore(ctx, a.Cfg.Database, a.Cfg.Restore, manifest)
	if err != nil {
		opErr = err
		return err
	}

	if _, err := io.Copy(restoreStream.Writer, compReader); err != nil {
		opErr = err
		return err
	}
	if err := restoreStream.Writer.Close(); err != nil {
		opErr = err
		return err
	}
	if err := restoreStream.Wait(); err != nil {
		opErr = err
		return err
	}
	return nil
}

func (a *App) Validate(ctx context.Context) error {
	if err := a.Adapter.Validate(ctx, a.Cfg.Database); err != nil {
		return err
	}
	prefix := util.BuildPrefix(a.Cfg.Storage.Prefix, a.Cfg.Database.Type, a.Cfg.Database.Database)
	_, err := a.Storage.List(ctx, prefix)
	return err
}

func (a *App) List(ctx context.Context) ([]storage.ObjectInfo, error) {
	prefix := util.BuildPrefix(a.Cfg.Storage.Prefix, a.Cfg.Database.Type, a.Cfg.Database.Database)
	return a.Storage.List(ctx, prefix)
}

func (a *App) writeManifest(ctx context.Context, manifest storage.Manifest) error {
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	key := storage.ManifestKey(manifest.Key)
	return a.Storage.Put(ctx, key, strings.NewReader(string(payload)), int64(len(payload)), map[string]string{"dbu-manifest": "true"})
}

func (a *App) readManifest(ctx context.Context, key string) (storage.Manifest, error) {
	manifestKey := storage.ManifestKey(key)
	reader, err := a.Storage.Get(ctx, manifestKey)
	if err != nil {
		return storage.Manifest{}, err
	}
	defer reader.Close()
	var manifest storage.Manifest
	if err := json.NewDecoder(reader).Decode(&manifest); err != nil {
		return storage.Manifest{}, err
	}
	return manifest, nil
}

func (a *App) applyRetention(ctx context.Context) error {
	policy := a.Cfg.Backup.RetentionPolicy
	if policy.KeepDays == 0 && policy.KeepLast == 0 && policy.MaxBytes == 0 {
		return nil
	}
	prefix := util.BuildPrefix(a.Cfg.Storage.Prefix, a.Cfg.Database.Type, a.Cfg.Database.Database)
	objects, err := a.Storage.List(ctx, prefix)
	if err != nil {
		return err
	}
	var backups []storage.ObjectInfo
	for _, obj := range objects {
		if obj.IsManifest {
			continue
		}
		backups = append(backups, obj)
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].Modified.After(backups[j].Modified) })

	cutoff := time.Now().AddDate(0, 0, -policy.KeepDays)
	var totalSize int64
	for _, obj := range backups {
		totalSize += obj.Size
	}
	for i, obj := range backups {
		if policy.KeepLast > 0 && i < policy.KeepLast {
			continue
		}
		if policy.KeepDays > 0 && obj.Modified.After(cutoff) {
			continue
		}
		if policy.MaxBytes > 0 && totalSize <= policy.MaxBytes {
			continue
		}
		_ = a.Storage.Delete(ctx, obj.Key)
		_ = a.Storage.Delete(ctx, storage.ManifestKey(obj.Key))
		totalSize -= obj.Size
	}
	return nil
}

func buildExtension(compression string, encryption bool) string {
	ext := "backup"
	switch compression {
	case compress.TypeGzip:
		ext += ".gz"
	case compress.TypeZstd:
		ext += ".zst"
	}
	if encryption {
		ext += ".enc"
	}
	return strings.TrimPrefix(ext, ".")
}

func statusFromErr(err error) string {
	if err == nil {
		return "success"
	}
	return "failed"
}
