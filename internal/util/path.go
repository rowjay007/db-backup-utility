package util

import (
	"fmt"
	"path"
	"strings"
	"time"
)

// BuildObjectKey constructs a normalized object key.
func BuildObjectKey(prefix, dbType, dbName, backupType string, when time.Time, extension string) string {
	parts := []string{}
	if prefix != "" {
		parts = append(parts, strings.Trim(prefix, "/"))
	}
	parts = append(parts, dbType, dbName)
	suffix := fmt.Sprintf("%s_%s", when.UTC().Format("20060102T150405Z"), backupType)
	if extension != "" {
		suffix = suffix + "." + extension
	}
	parts = append(parts, suffix)
	return path.Join(parts...)
}

// BuildPrefix builds the prefix for listing backups for a database.
func BuildPrefix(prefix, dbType, dbName string) string {
	parts := []string{}
	if prefix != "" {
		parts = append(parts, strings.Trim(prefix, "/"))
	}
	if dbType != "" {
		parts = append(parts, dbType)
	}
	if dbName != "" {
		parts = append(parts, dbName)
	}
	return path.Join(parts...)
}
