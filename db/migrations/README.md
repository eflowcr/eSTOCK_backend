# Database migrations

Versioned SQL migrations for eSTOCK. Uses [golang-migrate](https://github.com/golang-migrate/migrate).

## Naming

- **Up:** `NNNNNN_short_description.up.sql` (e.g. `000001_initial_schema.up.sql`)
- **Down:** `NNNNNN_short_description.down.sql` — must reverse the up migration

Run from `backend/` so paths resolve correctly.

## Makefile (from backend/)

| Target | Description |
|--------|-------------|
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Revert last migration |
| `make migrate-up-1` / `make migrate-down-1` | Apply or revert one migration |
| `make create-migration name=add_foo_table` | Create new migration pair (prompts for name if omitted) |
| `make migrate-force version=N` | Force schema version (fix broken state) |
| `make help` | Show all targets |

Requires `.env` with `DATABASE_URL` or `DB_SOURCE` (same as the app). See **eSTOCK doc/Roadmap - Migrations Backend** for full setup and conventions. For a full list of tables and columns, see **eSTOCK/Database Tables.md**.

## Install migrate CLI (for local/CI)

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

For SQL Server: `-tags 'sqlserver'` or build with both tags.
