package config

import "time"

// Config is the root configuration schema.
type Config struct {
	Global        GlobalConfig        `mapstructure:"global"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Backup        BackupConfig        `mapstructure:"backup"`
	Restore       RestoreConfig       `mapstructure:"restore"`
	Storage       StorageConfig       `mapstructure:"storage"`
	Notifications NotificationsConfig `mapstructure:"notifications"`
	Security      SecurityConfig      `mapstructure:"security"`
	Schedule      ScheduleConfig      `mapstructure:"schedule"`
}

type GlobalConfig struct {
	LogLevel          string        `mapstructure:"log_level"`
	LogFormat         string        `mapstructure:"log_format"` // json or console
	LockFile          string        `mapstructure:"lock_file"`
	OperationTimeout  time.Duration `mapstructure:"operation_timeout"`
	ConfigPassphrase  string        `mapstructure:"config_passphrase"` // optional; may come from env
	DisableTelemetry  bool          `mapstructure:"disable_telemetry"`
	UserAgent         string        `mapstructure:"user_agent"`
	AllowMissingTools bool          `mapstructure:"allow_missing_tools"`
}

type DatabaseConfig struct {
	Type              string            `mapstructure:"type"` // postgres, mysql, mongodb, sqlite
	Host              string            `mapstructure:"host"`
	Port              int               `mapstructure:"port"`
	Username          string            `mapstructure:"username"`
	Password          string            `mapstructure:"password"`
	Database          string            `mapstructure:"database"`
	Params            map[string]string `mapstructure:"params"`
	SSLMode           string            `mapstructure:"ssl_mode"`
	SSLCA             string            `mapstructure:"ssl_ca"`
	SSLCert           string            `mapstructure:"ssl_cert"`
	SSLKey            string            `mapstructure:"ssl_key"`
	ConnectionTimeout time.Duration     `mapstructure:"connection_timeout"`
	SQLitePath        string            `mapstructure:"sqlite_path"`
}

type BackupConfig struct {
	Type            string        `mapstructure:"type"`        // full, incremental, differential
	Compression     string        `mapstructure:"compression"` // none, gzip, zstd
	Encryption      bool          `mapstructure:"encryption"`
	EncryptionKey   string        `mapstructure:"encryption_key"`
	OutputPrefix    string        `mapstructure:"output_prefix"`
	RetryCount      int           `mapstructure:"retry_count"`
	RetryBackoff    time.Duration `mapstructure:"retry_backoff"`
	Idempotent      bool          `mapstructure:"idempotent"`
	MaxParallelism  int           `mapstructure:"max_parallelism"`
	Tables          []string      `mapstructure:"tables"`
	Collections     []string      `mapstructure:"collections"`
	IncludeSchema   bool          `mapstructure:"include_schema"`
	IncludeData     bool          `mapstructure:"include_data"`
	RetentionPolicy Retention     `mapstructure:"retention"`
}

type RestoreConfig struct {
	DryRun       bool     `mapstructure:"dry_run"`
	Tables       []string `mapstructure:"tables"`
	Collections  []string `mapstructure:"collections"`
	StopOnError  bool     `mapstructure:"stop_on_error"`
	DropExisting bool     `mapstructure:"drop_existing"`
}

type Retention struct {
	KeepLast int           `mapstructure:"keep_last"`
	KeepDays int           `mapstructure:"keep_days"`
	MaxBytes int64         `mapstructure:"max_bytes"`
	Schedule time.Duration `mapstructure:"schedule"`
}

type StorageConfig struct {
	Backend string     `mapstructure:"backend"` // local, s3
	Local   LocalStore `mapstructure:"local"`
	S3      S3Store    `mapstructure:"s3"`
	Prefix  string     `mapstructure:"prefix"`
	Tags    []string   `mapstructure:"tags"`
}

type LocalStore struct {
	Path string `mapstructure:"path"`
}

type S3Store struct {
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	AccessKey       string `mapstructure:"access_key"`
	SecretKey       string `mapstructure:"secret_key"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	ForcePathStyle  bool   `mapstructure:"force_path_style"`
	SessionToken    string `mapstructure:"session_token"`
	TLSInsecureSkip bool   `mapstructure:"tls_insecure_skip"`
}

type NotificationsConfig struct {
	Webhooks   []WebhookConfig  `mapstructure:"webhooks"`
	Mattermost []MattermostHook `mapstructure:"mattermost"`
	Matrix     []MatrixConfig   `mapstructure:"matrix"`
}

type WebhookConfig struct {
	Name    string            `mapstructure:"name"`
	URL     string            `mapstructure:"url"`
	Headers map[string]string `mapstructure:"headers"`
}

type MattermostHook struct {
	Name string `mapstructure:"name"`
	URL  string `mapstructure:"url"`
}

type MatrixConfig struct {
	Name        string `mapstructure:"name"`
	ServerURL   string `mapstructure:"server_url"`
	AccessToken string `mapstructure:"access_token"`
	RoomID      string `mapstructure:"room_id"`
}

type SecurityConfig struct {
	MinTLSVersion string `mapstructure:"min_tls_version"`
}

type ScheduleConfig struct {
	WindowStart string `mapstructure:"window_start"` // HH:MM local time
	WindowEnd   string `mapstructure:"window_end"`
	Timezone    string `mapstructure:"timezone"`
}
