# Contributing

Thanks for helping improve DBU.

## Development Setup

```bash
go mod tidy
go test ./...
```

## Guidelines

- Keep changes modular and well-tested
- Avoid vendor lock-in; use open standards and open-source tools
- Update docs for any new adapters or storage backends

## Adding a New Adapter

1. Implement `internal/db.Adapter`
2. Add tests for validation and pipeline compatibility
3. Update README and docs
