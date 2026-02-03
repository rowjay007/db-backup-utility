# Architecture Overview

DBU is built around three core abstractions:

- **Database adapters** (`internal/db`): encapsulate DB-specific backup and restore behavior
- **Storage backends** (`internal/storage`): define where artifacts are stored (local/S3)
- **Pipeline orchestration** (`internal/app`): handles streaming, compression, encryption, and retention

## Data Flow

Backup pipeline:

1. Adapter starts a dump stream (stdout or file reader)
2. Optional compression (`gzip` or `zstd`)
3. Optional streaming encryption (DARE)
4. Storage backend writes the stream (filesystem or S3)
5. Manifest is written alongside the backup artifact

Restore pipeline is the inverse:

1. Read from storage
2. Decrypt (if enabled)
3. Decompress (if enabled)
4. Stream into adapter restore process

## Extensibility

To add a new database engine:

1. Implement the `db.Adapter` interface
2. Register it in `db.NewAdapter`
3. Document any external tool dependencies

To add a new storage backend:

1. Implement the `storage.Storage` interface
2. Register it in `storage.New`

## Scheduling

DBU is scheduler-agnostic and designed to run via:

- `cron`
- `systemd` timers

Recommended pattern:

- Single cron entry per database
- Use `lock_file` to prevent overlapping runs
- Use retention policies for cleanup

## Observability

- JSON logs with structured fields
- Webhook-based notifications for success/failure
- Designed for integration with centralized logging systems
