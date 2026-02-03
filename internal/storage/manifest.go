package storage

import "time"

const ManifestSuffix = ".manifest.json"

type Manifest struct {
	ID           string    `json:"id"`
	Key          string    `json:"key"`
	DatabaseType string    `json:"database_type"`
	Database     string    `json:"database"`
	BackupType   string    `json:"backup_type"`
	Compression  string    `json:"compression"`
	Encryption   bool      `json:"encryption"`
	CreatedAt    time.Time `json:"created_at"`
	SizeBytes    int64     `json:"size_bytes"`
	Tables       []string  `json:"tables,omitempty"`
	Collections  []string  `json:"collections,omitempty"`
	ToolVersion  string    `json:"tool_version"`
}

func ManifestKey(objectKey string) string {
	return objectKey + ManifestSuffix
}
