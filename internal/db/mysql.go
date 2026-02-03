package db

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/rowjay/db-backup-utility/internal/config"
	"github.com/rowjay/db-backup-utility/internal/storage"
	"github.com/rowjay/db-backup-utility/internal/util"
)

type MySQLAdapter struct {
	allowMissingTools bool
}

func NewMySQLAdapter(allowMissingTools bool) *MySQLAdapter {
	return &MySQLAdapter{allowMissingTools: allowMissingTools}
}

func (m *MySQLAdapter) Name() string { return "mysql" }

func (m *MySQLAdapter) Capabilities() Capabilities {
	return Capabilities{Incremental: false, Differential: false, TableRestore: true}
}

func (m *MySQLAdapter) Validate(ctx context.Context, cfg config.DatabaseConfig) error {
	if !m.allowMissingTools {
		if err := util.RequireBinary("mysqldump"); err != nil {
			return err
		}
		if err := util.RequireBinary("mysql"); err != nil {
			return err
		}
	}

	if err := util.RequireBinary("mysqladmin"); err == nil {
		args := []string{"ping", "-h", cfg.Host, "-P", portOrDefault(cfg.Port, 3306), "-u", cfg.Username}
		if cfg.ConnectionTimeout > 0 {
			args = append(args, fmt.Sprintf("--connect-timeout=%d", int(cfg.ConnectionTimeout.Seconds())))
		}
		cmd := exec.CommandContext(ctx, "mysqladmin", args...)
		cmd.Env = util.MergeEnv(buildMySQLEnv(cfg))
		return cmd.Run()
	}
	return nil
}

func (m *MySQLAdapter) Dump(ctx context.Context, cfg config.DatabaseConfig, backup config.BackupConfig) (*DumpStream, error) {
	if !m.allowMissingTools {
		if err := util.RequireBinary("mysqldump"); err != nil {
			return nil, err
		}
	}
	if backup.Type != "" && backup.Type != "full" {
		return nil, fmt.Errorf("mysql does not support %s backups in this version", backup.Type)
	}

	args := []string{"--single-transaction", "--routines", "--events", "--triggers", "-h", cfg.Host, "-P", portOrDefault(cfg.Port, 3306), "-u", cfg.Username}
	if cfg.ConnectionTimeout > 0 {
		args = append(args, fmt.Sprintf("--connect-timeout=%d", int(cfg.ConnectionTimeout.Seconds())))
	}
	if cfg.SSLMode != "" {
		args = append(args, "--ssl-mode="+cfg.SSLMode)
	}
	if cfg.SSLCA != "" {
		args = append(args, "--ssl-ca="+cfg.SSLCA)
	}
	if cfg.SSLCert != "" {
		args = append(args, "--ssl-cert="+cfg.SSLCert)
	}
	if cfg.SSLKey != "" {
		args = append(args, "--ssl-key="+cfg.SSLKey)
	}

	if len(backup.Tables) > 0 {
		args = append(args, cfg.Database)
		args = append(args, backup.Tables...)
	} else {
		args = append(args, "--databases", cfg.Database)
	}

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	cmd.Env = util.MergeEnv(buildMySQLEnv(cfg))
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

func (m *MySQLAdapter) Restore(ctx context.Context, cfg config.DatabaseConfig, restore config.RestoreConfig, manifest storage.Manifest) (*RestoreStream, error) {
	if !m.allowMissingTools {
		if err := util.RequireBinary("mysql"); err != nil {
			return nil, err
		}
	}
	if len(restore.Tables) > 0 && len(manifest.Tables) == 0 {
		return nil, fmt.Errorf("selective table restore requires a backup created with specific tables")
	}
	if len(restore.Tables) > 0 && len(manifest.Tables) > 0 {
		allowed := make(map[string]struct{}, len(manifest.Tables))
		for _, t := range manifest.Tables {
			allowed[t] = struct{}{}
		}
		for _, t := range restore.Tables {
			if _, ok := allowed[t]; !ok {
				return nil, fmt.Errorf("table %s not present in backup", t)
			}
		}
	}
	args := []string{"-h", cfg.Host, "-P", portOrDefault(cfg.Port, 3306), "-u", cfg.Username, cfg.Database}
	if cfg.ConnectionTimeout > 0 {
		args = append(args, fmt.Sprintf("--connect-timeout=%d", int(cfg.ConnectionTimeout.Seconds())))
	}
	cmd := exec.CommandContext(ctx, "mysql", args...)
	cmd.Env = util.MergeEnv(buildMySQLEnv(cfg))
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

func buildMySQLEnv(cfg config.DatabaseConfig) []string {
	env := []string{}
	if cfg.Password != "" {
		env = append(env, "MYSQL_PWD="+cfg.Password)
	}
	return env
}
