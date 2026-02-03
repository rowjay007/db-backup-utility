package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/rowjay/db-backup-utility/internal/cryptoutil"
)

const (
	envPrefix = "DBU"
)

// Load reads configuration from a file (optionally encrypted), env vars, and defaults.
func Load(path string) (*Config, error) {
	vp := viper.New()
	vp.SetEnvPrefix(envPrefix)
	vp.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	vp.AutomaticEnv()

	setDefaults(vp)

	resolved, err := resolveConfigPath(path)
	if err != nil {
		return nil, err
	}

	if resolved != "" {
		data, readErr := os.ReadFile(resolved)
		if readErr != nil {
			return nil, fmt.Errorf("read config: %w", readErr)
		}
		if isEncryptedPath(resolved) {
			if typ := configTypeFromPath(resolved); typ != "" {
				vp.SetConfigType(typ)
			}
			key := os.Getenv("DBU_CONFIG_KEY")
			if key == "" {
				key = vp.GetString("global.config_passphrase")
			}
			if key == "" {
				return nil, errors.New("config file is encrypted but DBU_CONFIG_KEY is not set")
			}
			plain, decErr := decryptConfig(data, key)
			if decErr != nil {
				return nil, fmt.Errorf("decrypt config: %w", decErr)
			}
			if err := vp.ReadConfig(bytes.NewReader(plain)); err != nil {
				return nil, fmt.Errorf("parse config: %w", err)
			}
		} else {
			vp.SetConfigFile(resolved)
			if err := vp.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("read config: %w", err)
			}
		}
	}

	var cfg Config
	if err := vp.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	expandEnv(&cfg)
	applyPostLoadDefaults(&cfg)
	return &cfg, nil
}

func resolveConfigPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	if envPath := os.Getenv("DBU_CONFIG"); envPath != "" {
		return envPath, nil
	}

	candidates := []string{
		"dbu.yaml",
		"dbu.yml",
		"dbu.toml",
		"dbu.json",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	configDir, err := os.UserConfigDir()
	if err == nil {
		base := filepath.Join(configDir, "dbu")
		for _, c := range candidates {
			p := filepath.Join(base, c)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
		for _, c := range []string{"dbu.yaml.enc", "dbu.yml.enc", "dbu.toml.enc"} {
			p := filepath.Join(base, c)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return "", nil
}

func isEncryptedPath(path string) bool {
	return strings.HasSuffix(path, ".enc") || strings.HasSuffix(path, ".encrypted")
}

func configTypeFromPath(path string) string {
	switch {
	case strings.HasSuffix(path, ".toml") || strings.HasSuffix(path, ".toml.enc") || strings.HasSuffix(path, ".toml.encrypted"):
		return "toml"
	case strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".json.enc") || strings.HasSuffix(path, ".json.encrypted"):
		return "json"
	default:
		return "yaml"
	}
}

func setDefaults(vp *viper.Viper) {
	vp.SetDefault("global.log_level", "info")
	vp.SetDefault("global.log_format", "json")
	vp.SetDefault("global.operation_timeout", "2h")
	vp.SetDefault("backup.type", "full")
	vp.SetDefault("backup.compression", "zstd")
	vp.SetDefault("backup.retry_count", 3)
	vp.SetDefault("backup.retry_backoff", "10s")
	vp.SetDefault("backup.idempotent", true)
	vp.SetDefault("backup.include_schema", true)
	vp.SetDefault("backup.include_data", true)
	vp.SetDefault("storage.backend", "local")
	vp.SetDefault("storage.local.path", "./backups")
	vp.SetDefault("schedule.timezone", "")
}

func applyPostLoadDefaults(cfg *Config) {
	if cfg.Backup.RetryBackoff == 0 {
		cfg.Backup.RetryBackoff = 10 * time.Second
	}
	if cfg.Global.OperationTimeout == 0 {
		cfg.Global.OperationTimeout = 2 * time.Hour
	}
}

func expandEnv(cfg *Config) {
	cfg.Database.Password = os.ExpandEnv(cfg.Database.Password)
	cfg.Database.Username = os.ExpandEnv(cfg.Database.Username)
	cfg.Backup.EncryptionKey = os.ExpandEnv(cfg.Backup.EncryptionKey)
	cfg.Storage.S3.AccessKey = os.ExpandEnv(cfg.Storage.S3.AccessKey)
	cfg.Storage.S3.SecretKey = os.ExpandEnv(cfg.Storage.S3.SecretKey)
	cfg.Storage.S3.SessionToken = os.ExpandEnv(cfg.Storage.S3.SessionToken)
	cfg.Notifications = expandNotificationEnv(cfg.Notifications)
}

func expandNotificationEnv(cfg NotificationsConfig) NotificationsConfig {
	for i := range cfg.Webhooks {
		cfg.Webhooks[i].URL = os.ExpandEnv(cfg.Webhooks[i].URL)
	}
	for i := range cfg.Mattermost {
		cfg.Mattermost[i].URL = os.ExpandEnv(cfg.Mattermost[i].URL)
	}
	for i := range cfg.Matrix {
		cfg.Matrix[i].ServerURL = os.ExpandEnv(cfg.Matrix[i].ServerURL)
		cfg.Matrix[i].AccessToken = os.ExpandEnv(cfg.Matrix[i].AccessToken)
		cfg.Matrix[i].RoomID = os.ExpandEnv(cfg.Matrix[i].RoomID)
	}
	return cfg
}

func decryptConfig(ciphertext []byte, key string) ([]byte, error) {
	parsed, err := cryptoutil.ParseKey(key)
	if err != nil {
		return nil, err
	}
	return cryptoutil.DecryptConfig(ciphertext, parsed)
}
