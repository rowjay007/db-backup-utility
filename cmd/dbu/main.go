package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rowjay/db-backup-utility/internal/app"
	"github.com/rowjay/db-backup-utility/internal/config"
	"github.com/rowjay/db-backup-utility/internal/db"
	"github.com/rowjay/db-backup-utility/internal/logging"
	"github.com/rowjay/db-backup-utility/internal/notify"
	"github.com/rowjay/db-backup-utility/internal/storage"
	"github.com/rowjay/db-backup-utility/internal/util"
	"github.com/rowjay/db-backup-utility/internal/version"
)

type rootFlags struct {
	ConfigPath string
	LogLevel   string
	LogFormat  string
}

type overrideFlags struct {
	DBType        string
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBName        string
	SQLitePath    string
	Storage       string
	LocalPath     string
	S3Endpoint    string
	S3Bucket      string
	S3AccessKey   string
	S3SecretKey   string
	S3Region      string
	S3UseSSL      string
	S3PathStyle   string
	EncryptionKey string
}

func main() {
	root := &rootFlags{}
	overrides := &overrideFlags{}

	rootCmd := &cobra.Command{
		Use:   "dbu",
		Short: "Universal database backup and restore utility",
	}

	rootCmd.PersistentFlags().StringVar(&root.ConfigPath, "config", "", "Path to config file (yaml/toml/json or .enc)")
	rootCmd.PersistentFlags().StringVar(&root.LogLevel, "log-level", "", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&root.LogFormat, "log-format", "", "Log format (json, console)")

	rootCmd.PersistentFlags().StringVar(&overrides.DBType, "db-type", "", "Database type (postgres, mysql, mongodb, sqlite)")
	rootCmd.PersistentFlags().StringVar(&overrides.DBHost, "db-host", "", "Database host")
	rootCmd.PersistentFlags().IntVar(&overrides.DBPort, "db-port", 0, "Database port")
	rootCmd.PersistentFlags().StringVar(&overrides.DBUser, "db-user", "", "Database username")
	rootCmd.PersistentFlags().StringVar(&overrides.DBPassword, "db-password", "", "Database password")
	rootCmd.PersistentFlags().StringVar(&overrides.DBName, "db-name", "", "Database name")
	rootCmd.PersistentFlags().StringVar(&overrides.SQLitePath, "sqlite-path", "", "SQLite file path")

	rootCmd.PersistentFlags().StringVar(&overrides.Storage, "storage", "", "Storage backend (local, s3)")
	rootCmd.PersistentFlags().StringVar(&overrides.LocalPath, "storage-path", "", "Local storage path")
	rootCmd.PersistentFlags().StringVar(&overrides.S3Endpoint, "s3-endpoint", "", "S3 endpoint (MinIO/OSS)")
	rootCmd.PersistentFlags().StringVar(&overrides.S3Bucket, "s3-bucket", "", "S3 bucket")
	rootCmd.PersistentFlags().StringVar(&overrides.S3AccessKey, "s3-access-key", "", "S3 access key")
	rootCmd.PersistentFlags().StringVar(&overrides.S3SecretKey, "s3-secret-key", "", "S3 secret key")
	rootCmd.PersistentFlags().StringVar(&overrides.S3Region, "s3-region", "", "S3 region")
	rootCmd.PersistentFlags().StringVar(&overrides.S3UseSSL, "s3-ssl", "", "Use SSL for S3 endpoint (true/false)")
	rootCmd.PersistentFlags().StringVar(&overrides.S3PathStyle, "s3-path-style", "", "Force path-style S3 (true/false)")
	rootCmd.PersistentFlags().StringVar(&overrides.EncryptionKey, "encryption-key", "", "Encryption key (base64 or hex) for backups")

	rootCmd.AddCommand(newBackupCmd(root, overrides))
	rootCmd.AddCommand(newRestoreCmd(root, overrides))
	rootCmd.AddCommand(newValidateCmd(root, overrides))
	rootCmd.AddCommand(newListCmd(root, overrides))
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newBackupCmd(root *rootFlags, overrides *overrideFlags) *cobra.Command {
	backup := &cobra.Command{
		Use:   "backup",
		Short: "Create a backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(root, overrides)
			if err != nil {
				return err
			}
			logger := logging.Configure(cfg.Global.LogLevel, cfg.Global.LogFormat)
			adapter, err := db.NewAdapter(cfg.Database.Type, cfg.Global.AllowMissingTools)
			if err != nil {
				return err
			}
			store, err := storage.New(cfg.Storage)
			if err != nil {
				return err
			}
			appSvc := app.New(cfg, adapter, store, logger, notify.FromConfig(cfg.Notifications))

			ctx, cancel := context.WithTimeout(context.Background(), cfg.Global.OperationTimeout)
			defer cancel()

			return util.Retry(ctx, cfg.Backup.RetryCount, cfg.Backup.RetryBackoff, func() error {
				res, err := appSvc.Backup(ctx)
				if err != nil {
					return err
				}
				logger.Info().Str("key", res.Key).Int64("size", res.Manifest.SizeBytes).Msg("backup completed")
				return nil
			})
		},
	}
	backup.Flags().StringSliceVar(&overridesDBTables, "tables", nil, "Tables to include (PG/MySQL)")
	backup.Flags().StringSliceVar(&overridesDBCollections, "collections", nil, "Collections to include (MongoDB)")
	backup.Flags().StringVar(&backupType, "type", "", "Backup type (full/incremental/differential)")
	backup.Flags().StringVar(&backupCompression, "compression", "", "Compression (none/gzip/zstd)")
	backup.Flags().BoolVar(&backupEncryption, "encrypt", false, "Enable encryption")
	backup.Flags().IntVar(&backupRetry, "retry", 0, "Retry attempts")
	backup.Flags().DurationVar(&backupRetryBackoff, "retry-backoff", 0, "Retry backoff")
	return backup
}

var (
	overridesDBTables      []string
	overridesDBCollections []string
	backupType             string
	backupCompression      string
	backupEncryption       bool
	backupRetry            int
	backupRetryBackoff     time.Duration
)

func newRestoreCmd(root *rootFlags, overrides *overrideFlags) *cobra.Command {
	var key string
	var dryRun bool
	var tables []string
	var collections []string
	var dropExisting bool

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" {
				return fmt.Errorf("--key is required")
			}
			cfg, err := loadConfig(root, overrides)
			if err != nil {
				return err
			}
			if dryRun {
				cfg.Restore.DryRun = true
			}
			if len(tables) > 0 {
				cfg.Restore.Tables = tables
			}
			if len(collections) > 0 {
				cfg.Restore.Collections = collections
			}
			cfg.Restore.DropExisting = dropExisting

			logger := logging.Configure(cfg.Global.LogLevel, cfg.Global.LogFormat)
			adapter, err := db.NewAdapter(cfg.Database.Type, cfg.Global.AllowMissingTools)
			if err != nil {
				return err
			}
			store, err := storage.New(cfg.Storage)
			if err != nil {
				return err
			}
			appSvc := app.New(cfg, adapter, store, logger, notify.FromConfig(cfg.Notifications))

			ctx, cancel := context.WithTimeout(context.Background(), cfg.Global.OperationTimeout)
			defer cancel()

			if err := appSvc.Restore(ctx, key); err != nil {
				return err
			}
			logger.Info().Str("key", key).Msg("restore completed")
			return nil
		},
	}

	cmd.Flags().StringVar(&key, "key", "", "Backup object key to restore")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run")
	cmd.Flags().StringSliceVar(&tables, "tables", nil, "Tables to restore")
	cmd.Flags().StringSliceVar(&collections, "collections", nil, "Collections to restore")
	cmd.Flags().BoolVar(&dropExisting, "drop-existing", false, "Drop existing objects before restore")

	return cmd
}

func newValidateCmd(root *rootFlags, overrides *overrideFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration and connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(root, overrides)
			if err != nil {
				return err
			}
			logger := logging.Configure(cfg.Global.LogLevel, cfg.Global.LogFormat)
			adapter, err := db.NewAdapter(cfg.Database.Type, cfg.Global.AllowMissingTools)
			if err != nil {
				return err
			}
			store, err := storage.New(cfg.Storage)
			if err != nil {
				return err
			}
			appSvc := app.New(cfg, adapter, store, logger, notify.FromConfig(cfg.Notifications))
			ctx, cancel := context.WithTimeout(context.Background(), cfg.Global.OperationTimeout)
			defer cancel()
			if err := appSvc.Validate(ctx); err != nil {
				return err
			}
			logger.Info().Msg("validation succeeded")
			return nil
		},
	}
}

func newListCmd(root *rootFlags, overrides *overrideFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(root, overrides)
			if err != nil {
				return err
			}
			logger := logging.Configure(cfg.Global.LogLevel, cfg.Global.LogFormat)
			adapter, err := db.NewAdapter(cfg.Database.Type, cfg.Global.AllowMissingTools)
			if err != nil {
				return err
			}
			store, err := storage.New(cfg.Storage)
			if err != nil {
				return err
			}
			appSvc := app.New(cfg, adapter, store, logger, notify.FromConfig(cfg.Notifications))
			ctx, cancel := context.WithTimeout(context.Background(), cfg.Global.OperationTimeout)
			defer cancel()
			items, err := appSvc.List(ctx)
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Printf("%s\t%d\t%s\n", item.Key, item.Size, item.Modified.Format(time.RFC3339))
			}
			logger.Info().Msg("list completed")
			return nil
		},
	}
}

func newConfigCmd() *cobra.Command {
	var input string
	var output string
	var key string

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Config utilities",
	}

	encrypt := &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt a config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if input == "" || output == "" || key == "" {
				return fmt.Errorf("--input, --output, and --key are required")
			}
			return config.EncryptConfigFile(input, output, key)
		},
	}
	encrypt.Flags().StringVar(&input, "input", "", "Input config file")
	encrypt.Flags().StringVar(&output, "output", "", "Output encrypted config file")
	encrypt.Flags().StringVar(&key, "key", "", "Encryption key (base64 or hex)")

	cmd.AddCommand(encrypt)
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dbu %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
		},
	}
}

func loadConfig(root *rootFlags, overrides *overrideFlags) (*config.Config, error) {
	cfg, err := config.Load(root.ConfigPath)
	if err != nil {
		return nil, err
	}
	applyOverrides(cfg, root, overrides)
	return cfg, nil
}

func applyOverrides(cfg *config.Config, root *rootFlags, overrides *overrideFlags) {
	if root.LogLevel != "" {
		cfg.Global.LogLevel = root.LogLevel
	}
	if root.LogFormat != "" {
		cfg.Global.LogFormat = root.LogFormat
	}

	if overrides.DBType != "" {
		cfg.Database.Type = overrides.DBType
	}
	if overrides.DBHost != "" {
		cfg.Database.Host = overrides.DBHost
	}
	if overrides.DBPort != 0 {
		cfg.Database.Port = overrides.DBPort
	}
	if overrides.DBUser != "" {
		cfg.Database.Username = overrides.DBUser
	}
	if overrides.DBPassword != "" {
		cfg.Database.Password = overrides.DBPassword
	}
	if overrides.DBName != "" {
		cfg.Database.Database = overrides.DBName
	}
	if overrides.SQLitePath != "" {
		cfg.Database.SQLitePath = overrides.SQLitePath
	}

	if overrides.Storage != "" {
		cfg.Storage.Backend = overrides.Storage
	}
	if overrides.LocalPath != "" {
		cfg.Storage.Local.Path = overrides.LocalPath
	}
	if overrides.S3Endpoint != "" {
		cfg.Storage.S3.Endpoint = overrides.S3Endpoint
	}
	if overrides.S3Bucket != "" {
		cfg.Storage.S3.Bucket = overrides.S3Bucket
	}
	if overrides.S3AccessKey != "" {
		cfg.Storage.S3.AccessKey = overrides.S3AccessKey
	}
	if overrides.S3SecretKey != "" {
		cfg.Storage.S3.SecretKey = overrides.S3SecretKey
	}
	if overrides.S3Region != "" {
		cfg.Storage.S3.Region = overrides.S3Region
	}
	if overrides.S3Endpoint != "" {
		cfg.Storage.S3.Endpoint = overrides.S3Endpoint
	}
	if overrides.S3UseSSL != "" {
		cfg.Storage.S3.UseSSL = strings.EqualFold(overrides.S3UseSSL, "true") || overrides.S3UseSSL == "1"
	}
	if overrides.S3PathStyle != "" {
		cfg.Storage.S3.ForcePathStyle = strings.EqualFold(overrides.S3PathStyle, "true") || overrides.S3PathStyle == "1"
	}

	if overrides.EncryptionKey != "" {
		cfg.Backup.EncryptionKey = overrides.EncryptionKey
	}

	if backupType != "" {
		cfg.Backup.Type = strings.ToLower(backupType)
	}
	if backupCompression != "" {
		cfg.Backup.Compression = strings.ToLower(backupCompression)
	}
	if backupEncryption {
		cfg.Backup.Encryption = true
	}
	if backupRetry > 0 {
		cfg.Backup.RetryCount = backupRetry
	}
	if backupRetryBackoff > 0 {
		cfg.Backup.RetryBackoff = backupRetryBackoff
	}
	if len(overridesDBTables) > 0 {
		cfg.Backup.Tables = overridesDBTables
	}
	if len(overridesDBCollections) > 0 {
		cfg.Backup.Collections = overridesDBCollections
	}

	cfg.Database.Type = strings.ToLower(cfg.Database.Type)
	cfg.Backup.Type = strings.ToLower(cfg.Backup.Type)
	cfg.Backup.Compression = strings.ToLower(cfg.Backup.Compression)
	cfg.Storage.Backend = strings.ToLower(cfg.Storage.Backend)
}
