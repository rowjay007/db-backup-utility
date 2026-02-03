# Universal Database Backup Utility (DBU)

DBU is a cross-platform, production-grade CLI for backing up and restoring multiple databases using a modular adapter architecture. It is written in Go, uses open-source components only, and is designed for enterprise environments with strong security, reliability, and observability needs.

## Highlights

- Modular adapters for PostgreSQL, MySQL/MariaDB, MongoDB, and SQLite
- Streaming backup/restore pipelines for large datasets
- Compression (gzip, zstd) and streaming encryption (DARE)
- Pluggable storage backends: local filesystem and S3-compatible (MinIO/Ceph/Swift)
- Structured JSON logging and webhook notifications
- Cross-platform support for Linux, macOS, and Windows

## Quick Start

Build the CLI:

```bash
go build -o dbu ./cmd/dbu
```

Validate configuration:

```bash
./dbu validate --config examples/config.yaml
```

Run a backup:

```bash
./dbu backup --config examples/config.yaml
```

Restore a backup:

```bash
./dbu restore --config examples/config.yaml --key backups/postgres/appdb/20240101T100000Z_full.backup.zst.enc
```

## Configuration

DBU supports configuration via YAML/TOML/JSON, environment variables, and CLI flags. Environment variables are prefixed with `DBU_` and use `_` for nesting (example: `DBU_DATABASE_HOST`).

See `examples/config.yaml` for a full example.

### Encrypted Config Files

To encrypt a config file (AES-256 DARE):

```bash
./dbu config encrypt --input examples/config.yaml --output examples/config.yaml.enc --key base64:YOUR_BASE64_KEY
```

Then set `DBU_CONFIG_KEY` when running DBU:

```bash
export DBU_CONFIG=examples/config.yaml.enc
export DBU_CONFIG_KEY=base64:YOUR_BASE64_KEY
./dbu backup
```

Encryption keys must be 32 bytes, provided as base64 or hex (prefix with `base64:` or `hex:`).

## Supported Databases

- PostgreSQL (primary reference, Neon compatible)
- MySQL / MariaDB
- MongoDB
- SQLite

Adapters rely on vendor CLI tools. Install the appropriate client tools for the database you are backing up:

- PostgreSQL: `pg_dump`, `pg_restore`, `pg_isready`
- MySQL/MariaDB: `mysqldump`, `mysql`, `mysqladmin`
- MongoDB: `mongodump`, `mongorestore`, `mongosh`

SQLite uses file streaming by default.

## Storage Backends

- Local filesystem (default)
- S3-compatible object storage via MinIO SDK (MinIO, Ceph, OpenStack Swift, etc.)

## Scheduling

DBU is designed to work with external schedulers:

- `cron`
- `systemd` timers

See `docs/ARCHITECTURE.md` for suggested patterns.

## Notifications

Supported notification channels:

- Generic webhooks (JSON payload)
- Mattermost incoming webhooks
- Matrix (client-server API)

## Documentation

- `docs/ARCHITECTURE.md`
- `docs/NEON.md`
- `docs/CONTRIBUTING.md`

## Security & Compliance Notes

- Encrypted backups at rest with streaming AEAD
- Credentials can be provided via environment variables or encrypted config
- JSON logs suitable for audit trails (SOC2/ISO aligned)

## License

This project is open-source and intended for enterprise use. Add your preferred license file if required.
# db-backup-utility
