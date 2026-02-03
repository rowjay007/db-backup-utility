package db

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rowjay/db-backup-utility/internal/config"
	"github.com/rowjay/db-backup-utility/internal/storage"
	"github.com/rowjay/db-backup-utility/internal/util"
)

type MongoAdapter struct {
	allowMissingTools bool
}

func NewMongoAdapter(allowMissingTools bool) *MongoAdapter {
	return &MongoAdapter{allowMissingTools: allowMissingTools}
}

func (m *MongoAdapter) Name() string { return "mongodb" }

func (m *MongoAdapter) Capabilities() Capabilities {
	return Capabilities{Incremental: false, Differential: false, CollectionRestore: true}
}

func (m *MongoAdapter) Validate(ctx context.Context, cfg config.DatabaseConfig) error {
	if !m.allowMissingTools {
		if err := util.RequireBinary("mongodump"); err != nil {
			return err
		}
		if err := util.RequireBinary("mongorestore"); err != nil {
			return err
		}
	}
	if err := util.RequireBinary("mongosh"); err == nil {
		cmd := exec.CommandContext(ctx, "mongosh", "--quiet", "--eval", "db.runCommand({ ping: 1 })")
		cmd.Env = util.MergeEnv(buildMongoEnv(cfg))
		return cmd.Run()
	}
	return nil
}

func (m *MongoAdapter) Dump(ctx context.Context, cfg config.DatabaseConfig, backup config.BackupConfig) (*DumpStream, error) {
	if !m.allowMissingTools {
		if err := util.RequireBinary("mongodump"); err != nil {
			return nil, err
		}
	}
	if backup.Type != "" && backup.Type != "full" {
		return nil, fmt.Errorf("mongodb does not support %s backups in this version", backup.Type)
	}
	args := []string{"--archive", "--db", cfg.Database}
	args = append(args, mongoConnArgs(cfg)...)
	for _, coll := range backup.Collections {
		args = append(args, "--collection", coll)
	}
	cmd := exec.CommandContext(ctx, "mongodump", args...)
	cmd.Env = util.MergeEnv(buildMongoEnv(cfg))
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

func (m *MongoAdapter) Restore(ctx context.Context, cfg config.DatabaseConfig, restore config.RestoreConfig, manifest storage.Manifest) (*RestoreStream, error) {
	if !m.allowMissingTools {
		if err := util.RequireBinary("mongorestore"); err != nil {
			return nil, err
		}
	}
	args := []string{"--archive", "--db", cfg.Database}
	args = append(args, mongoConnArgs(cfg)...)
	if restore.DropExisting {
		args = append(args, "--drop")
	}
	for _, coll := range restore.Collections {
		args = append(args, "--nsInclude", fmt.Sprintf("%s.%s", cfg.Database, coll))
	}
	cmd := exec.CommandContext(ctx, "mongorestore", args...)
	cmd.Env = util.MergeEnv(buildMongoEnv(cfg))
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

func mongoConnArgs(cfg config.DatabaseConfig) []string {
	args := []string{}
	if cfg.Host != "" {
		args = append(args, "--host", cfg.Host)
	}
	if cfg.Port != 0 {
		args = append(args, "--port", fmt.Sprintf("%d", cfg.Port))
	}
	if cfg.Username != "" {
		args = append(args, "--username", cfg.Username)
	}
	if cfg.Password != "" {
		args = append(args, "--password", cfg.Password)
	}
	if cfg.SSLMode != "" {
		if strings.EqualFold(cfg.SSLMode, "disable") {
			// no tls
		} else {
			args = append(args, "--tls")
		}
	}
	if cfg.SSLCA != "" {
		args = append(args, "--tlsCAFile", cfg.SSLCA)
	}
	if cfg.SSLCert != "" {
		args = append(args, "--tlsCertificateKeyFile", cfg.SSLCert)
	}
	if cfg.Params != nil {
		if authSource, ok := cfg.Params["authSource"]; ok {
			args = append(args, "--authenticationDatabase", authSource)
		}
	}
	return args
}

func buildMongoEnv(cfg config.DatabaseConfig) []string {
	env := []string{}
	if uri, ok := cfg.Params["uri"]; ok && uri != "" {
		env = append(env, "MONGODB_URI="+uri)
	}
	return env
}
