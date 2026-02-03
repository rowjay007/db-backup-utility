package util

import (
	"strings"
	"testing"
	"time"
)

func TestBuildObjectKey(t *testing.T) {
	when := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	key := BuildObjectKey("backups", "postgres", "appdb", "full", when, "backup.zst")
	if !strings.HasPrefix(key, "backups/postgres/appdb/") {
		t.Fatalf("unexpected prefix: %s", key)
	}
	if !strings.Contains(key, "_full.backup.zst") {
		t.Fatalf("unexpected suffix: %s", key)
	}
}

func TestBuildPrefix(t *testing.T) {
	prefix := BuildPrefix("backups", "postgres", "appdb")
	if prefix != "backups/postgres/appdb" {
		t.Fatalf("unexpected prefix: %s", prefix)
	}
}
