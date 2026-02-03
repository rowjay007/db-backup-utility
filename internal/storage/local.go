package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Local struct {
	BasePath string
}

func NewLocal(path string) *Local {
	return &Local{BasePath: path}
}

func (l *Local) Put(ctx context.Context, key string, reader io.Reader, _ int64, _ map[string]string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	target := filepath.Join(l.BasePath, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return err
	}
	return nil
}

func (l *Local) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return os.Open(filepath.Join(l.BasePath, filepath.FromSlash(key)))
}

func (l *Local) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	select {
	case <-ctx.Done():
		return ObjectInfo{}, ctx.Err()
	default:
	}
	path := filepath.Join(l.BasePath, filepath.FromSlash(key))
	info, err := os.Stat(path)
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Key: key, Size: info.Size(), Modified: info.ModTime(), IsManifest: strings.HasSuffix(key, ManifestSuffix)}, nil
}

func (l *Local) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	root := filepath.Join(l.BasePath, filepath.FromSlash(prefix))
	infos := []ObjectInfo{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(l.BasePath, path)
		if relErr != nil {
			return nil
		}
		stat, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		key := filepath.ToSlash(rel)
		isManifest := strings.HasSuffix(key, ManifestSuffix)
		infos = append(infos, ObjectInfo{Key: key, Size: stat.Size(), Modified: stat.ModTime(), IsManifest: isManifest})
		return nil
	})

	return infos, nil
}

func (l *Local) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return os.Remove(filepath.Join(l.BasePath, filepath.FromSlash(key)))
}

func (l *Local) Exists(ctx context.Context, key string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	_, err := os.Stat(filepath.Join(l.BasePath, filepath.FromSlash(key)))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (l *Local) CleanupOld(ctx context.Context, prefix string, cutoff time.Time, keep int) ([]ObjectInfo, error) {
	objects, err := l.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	eligible := []ObjectInfo{}
	for _, obj := range objects {
		if obj.Modified.Before(cutoff) && !obj.IsManifest {
			eligible = append(eligible, obj)
		}
	}
	if keep > 0 && len(eligible) > keep {
		eligible = eligible[:len(eligible)-keep]
	}
	for _, obj := range eligible {
		_ = l.Delete(ctx, obj.Key)
		_ = l.Delete(ctx, ManifestKey(obj.Key))
	}
	return eligible, nil
}
