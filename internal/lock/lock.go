package lock

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

type Lock struct {
	file *flock.Flock
}

// Acquire obtains a filesystem lock to prevent overlapping operations.
func Acquire(path string) (*Lock, error) {
	if path == "" {
		path = filepath.Join(os.TempDir(), "dbu.lock")
	}
	lock := flock.New(path)
	ok, err := lock.TryLock()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("another backup/restore is already running (lock: %s)", path)
	}
	return &Lock{file: lock}, nil
}

// Release frees the lock.
func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Unlock()
}
