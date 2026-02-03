package db

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/rowjay/db-backup-utility/internal/config"
	"github.com/rowjay/db-backup-utility/internal/storage"
	"github.com/rowjay/db-backup-utility/internal/util"
)

type PostgresAdapter struct {
	allowMissingTools bool
}

func NewPostgresAdapter(allowMissingTools bool) *PostgresAdapter {
	return &PostgresAdapter{allowMissingTools: allowMissingTools}
}

func (p *PostgresAdapter) Name() string { return "postgres" }

func (p *PostgresAdapter) Capabilities() Capabilities {
	return Capabilities{Incremental: false, Differential: false, TableRestore: true}
}

func (p *PostgresAdapter) Validate(ctx context.Context, cfg config.DatabaseConfig) error {
	if !p.allowMissingTools {
		if err := util.RequireBinary("pg_dump"); err != nil {
			return err
		}
		if err := util.RequireBinary("pg_restore"); err != nil {
			return err
		}
	}

	if err := util.RequireBinary("pg_isready"); err == nil {
		cmd := exec.CommandContext(ctx, "pg_isready", "-h", cfg.Host, "-p", portOrDefault(cfg.Port, 5432), "-U", cfg.Username, "-d", cfg.Database)
		cmd.Env = util.MergeEnv(buildPostgresEnv(cfg))
		return cmd.Run()
	}

	if err := util.RequireBinary("psql"); err != nil {
		return nil
	}
	cmd := exec.CommandContext(ctx, "psql", "-c", "SELECT 1")
	cmd.Env = util.MergeEnv(buildPostgresEnv(cfg))
	return cmd.Run()
}

func (p *PostgresAdapter) Dump(ctx context.Context, cfg config.DatabaseConfig, backup config.BackupConfig) (*DumpStream, error) {
	if !p.allowMissingTools {
		if err := util.RequireBinary("pg_dump"); err != nil {
			return nil, err
		}
	}
	if backup.Type != "" && backup.Type != "full" {
		return nil, fmt.Errorf("postgres does not support %s backups in this version", backup.Type)
	}

	args := []string{"--format=custom", "--no-owner", "--no-privileges"}
	if backup.IncludeSchema && !backup.IncludeData {
		args = append(args, "--schema-only")
	}
	if backup.IncludeData && !backup.IncludeSchema {
		args = append(args, "--data-only")
	}
	for _, tbl := range backup.Tables {
		args = append(args, "--table", tbl)
	}
	args = append(args, cfg.Database)

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = util.MergeEnv(buildPostgresEnv(cfg))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = stderrSink()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &DumpStream{Reader: stdout, Wait: cmd.Wait}, nil
}

func (p *PostgresAdapter) Restore(ctx context.Context, cfg config.DatabaseConfig, restore config.RestoreConfig, manifest storage.Manifest) (*RestoreStream, error) {
	if !p.allowMissingTools {
		if err := util.RequireBinary("pg_restore"); err != nil {
			return nil, err
		}
	}
	args := []string{"--dbname", cfg.Database, "--no-owner", "--no-privileges"}
	if restore.DropExisting {
		args = append(args, "--clean", "--if-exists")
	}
	if restore.StopOnError {
		args = append(args, "--exit-on-error")
	}
	for _, tbl := range restore.Tables {
		args = append(args, "--table", tbl)
	}
	cmd := exec.CommandContext(ctx, "pg_restore", args...)
	cmd.Env = util.MergeEnv(buildPostgresEnv(cfg))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = stderrSink()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &RestoreStream{Writer: stdin, Wait: cmd.Wait}, nil
}

func buildPostgresEnv(cfg config.DatabaseConfig) []string {
	env := []string{
		"PGHOST=" + cfg.Host,
		"PGPORT=" + portOrDefault(cfg.Port, 5432),
		"PGUSER=" + cfg.Username,
		"PGDATABASE=" + cfg.Database,
	}
	if cfg.Password != "" {
		env = append(env, "PGPASSWORD="+cfg.Password)
	}
	if cfg.SSLMode != "" {
		env = append(env, "PGSSLMODE="+cfg.SSLMode)
	}
	if cfg.SSLCA != "" {
		env = append(env, "PGSSLROOTCERT="+cfg.SSLCA)
	}
	if cfg.SSLCert != "" {
		env = append(env, "PGSSLCERT="+cfg.SSLCert)
	}
	if cfg.SSLKey != "" {
		env = append(env, "PGSSLKEY="+cfg.SSLKey)
	}
	if cfg.ConnectionTimeout > 0 {
		env = append(env, "PGCONNECT_TIMEOUT="+strconv.Itoa(int(cfg.ConnectionTimeout.Seconds())))
	}
	return env
}

func portOrDefault(port int, def int) string {
	if port == 0 {
		return strconv.Itoa(def)
	}
	return strconv.Itoa(port)
}
