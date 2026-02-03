# Neon PostgreSQL Setup Guide

Neon is a PostgreSQL-compatible hosted database and can be used for development and testing with DBU.

## Steps

1. Create a Neon project and database
2. Obtain the connection details (host, port, database, username, password)
3. Add those values to your DBU config (see `examples/config.yaml`)

Example snippet:

```yaml
database:
  type: postgres
  host: your-project.neon.tech
  port: 5432
  username: neon_user
  password: ${NEON_PASSWORD}
  database: neon_db
  ssl_mode: require
```

## Notes

- Ensure `pg_dump`, `pg_restore`, and `pg_isready` are installed locally.
- For large datasets, prefer `zstd` compression and streaming encryption.
