# eSTOCK Backend

Stack: Go 1.22+ · Gin · GORM · sqlc · PostgreSQL 16 · Redis (optional)

## Features (post-Sprint S1)

- JWT auth + RBAC con role-based permissions
- **Forgot password flow** (S1): rate-limited (5/h/IP), audit log, invalidación de sesiones al reset, Argon2+AES encryption
- Articles + Inventory CRUD con `reserved_qty` + `available_qty` (S1)
- **Cross-location picking** (S1): `Allocations []LocationAllocation`, FEFO cross-location greedy, lazy reservations al `StartPickingTask`
- Receiving tasks con lotes estructurados (`LotEntry`) + upsert automático + unique index por SKU
- **State machines** formalizadas (picking: open→assigned→in_progress→completed; receiving: open→in_progress→completed)
- **Cron unificado** (S1): stock alerts + stale reservations cleanup + `pg_advisory_lock` + admin trigger endpoint
- Validaciones cross-module: transfers y adjustments bloqueados si afectan `reserved_qty`
- Parser legacy retrocompat (S1) para tasks pre-sprint sin `allocations`
- Audit log fire-and-forget
- Excel import/export (articles, inventory, picking tasks)
- API docs auto-generados en dev: route list, OpenAPI spec, Swagger UI

## Quick start

### Prerequisites

- Go 1.22+
- PostgreSQL 16
- `golang-migrate` CLI: `brew install golang-migrate`
- `sqlc` (solo si tocas queries/migrations): `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

### Setup

```bash
cp .env.example .env   # editar con valores locales
go mod download
make migrate-up
make server            # :8080
```

### Env vars requeridas

| Var | Descripción |
|---|---|
| `DATABASE_URL` o `DB_*` | Postgres connection string (ver .env.example para ambas formas) |
| `JWT_SECRET` | Firma JWT + deriva key Argon2 para passwords (min 32 chars) |
| `APP_URL` | URL pública del frontend — usada para generar links de reset password |
| `ENVIRONMENT` | `development` \| `release` (Gin mode + log format) |
| `RESEND_API_KEY` | Opcional — si no se setea, emails van a stdout (LoggerEmailSender) |

### Make targets disponibles

```bash
make server              # go run cmd/main.go
make migrate-up          # aplica todas las migraciones pendientes
make migrate-down-1      # revierte última migración
make create-migration name=my_change  # crea up/down SQL vacíos
make sqlc                # regenera db/sqlc/ desde db/query/*.sql
make test                # go test ./... -count=1
make test-unit           # idem con -short (sin Docker)
make test-articles-integration  # integration tests (requiere Docker)
make docs                # muestra URLs de API docs (correr make server primero)
```

## API Endpoints

Todos bajo `/api/`. Ver docs completos: `GET /api/docs/routes` (dev only).

### Auth (`/api/auth`)

| Método | Path | Notas |
|---|---|---|
| POST | `/login` | |
| POST | `/forgot-password` | **S1** — rate limit 5/h/IP |
| POST | `/reset-password` | **S1** — rate limit 10/h/IP |

### Picking Tasks (`/api/picking-tasks`)

| Método | Path | Notas |
|---|---|---|
| GET | `/` | |
| GET | `/:id` | |
| POST | `/` | |
| PUT | `/:id` | |
| PATCH | `/:id/start` | **S1** — aplica lazy reservation |
| PATCH | `/:id/cancel` | |
| PATCH | `/:id/complete` | signature S1: `allocations[]`, no `location` string |
| PATCH | `/:id/complete-line` | |
| GET | `/import/template` | |
| POST | `/import` | Excel |
| GET | `/export` | Excel |

### Inventory (`/api/inventory`)

| Método | Path | Notas |
|---|---|---|
| GET | `/` | incluye `reserved_qty` + `available_qty` (S1) |
| GET | `/pick-suggestions/:sku?qty=N` | **contrato S1**: responde `PickSuggestionResponse` |
| GET | `/sku/:sku/location/:location` | |
| POST | `/` | |
| PATCH | `/id/:id` | |
| DELETE | `/id/:id/:location` | |
| GET/POST/DELETE | `/id/:id/lots` | |
| GET/POST/DELETE | `/id/:id/serials` | |
| GET | `/sku/:sku/trend` | |

### Receiving Tasks (`/api/receiving-tasks`)

CRUD completo + `PATCH /:id/complete` + `PATCH /:id/complete-line`.
Lifecycle: `open → in_progress → completed | completed_with_differences`.

### Admin (`/api/admin/cron`) — requiere permiso `cron:trigger`

| Método | Path | Notas |
|---|---|---|
| POST | `/trigger?job=stock_alerts\|stale_reservations\|all` | **S1** |

### Otros grupos de endpoints

`/articles`, `/locations`, `/location-types`, `/lots`, `/serials`, `/stock-alerts`, `/stock-transfers`, `/adjustments`, `/adjustment-reason-codes`, `/users`, `/roles`, `/audit-logs`, `/dashboard`, `/gamification`, `/presentations`, `/presentation-types`, `/presentation-conversions`, `/inventory_movements`, `/user` (preferences).

## Database

### Migrations

- `000001–000016`: schema baseline pre-S1
- `000017` (S1): `reserved_qty` en `inventory`, tabla `password_reset_tokens`, unique index en `lots` (sku + lot_number donde status ≠ archived)

```bash
make migrate-up      # aplica todo
make migrate-down-1  # revierte una (safe — pide confirmación)
```

### Gotchas críticos

- **Password hashing:** `tools.Encrypt(plaintext, JWT_SECRET)` — Argon2+AES. NO usar bcrypt.
- **PKs string:** SIEMPRE llamar `tools.GenerateNanoid(tx)` antes de `tx.Create()` en cualquier struct con `ID string`. Sin esto GORM inserta `id=''`, corrompiendo la fila silenciosamente. Afecta: `Lot`, `Inventory`, `InventoryLot`, `InventoryMovement`, y cualquier nuevo struct con PK string.
- **sqlc drift:** `db/sqlc/models.go` debe regenerarse con `make sqlc` en cualquier PR que toque migrations o queries. El CI `backend-sqlc` fallará si hay drift.

## Tests

```bash
go test ./... -count=1          # todos los tests
go test ./... -short -count=1   # unit only (sin Docker)
go test ./tools/... -v          # integration (requiere Docker + testcontainers)
```

CI corre `go test ./... -short` en cada push a `dev`. Integration tests corren en el job `backend-sqlc`.

## Deploy

Para el deploy de S1 a producción, seguir el playbook completo:
`~/Documents/obsidian/Jafet/Projects/ePRAC/eSTOCK/plans/2026-04-16-sprint-s1-deploy-playbook.md`

**Orden crítico:** backend PRIMERO (migración 000017 + deploy), luego frontend.

### Antes del deploy S1 (limpieza de datos huérfanos)

```sql
DELETE FROM inventory_movements WHERE id = '';
DELETE FROM inventory_lots WHERE lot_id = '' OR inventory_id = '';
DELETE FROM lots WHERE id = '';
DELETE FROM inventory WHERE id = '';
```

## CI/CD

Ver `DEVELOPMENT.md` para detalle completo de workflows y branch strategy.

| Workflow | Cuándo corre | Qué hace |
|---|---|---|
| `deploy-dev.yml` | Push a `dev` | Tests → docker build → push `:dev` → rolling update |
| `deploy-prod.yml` | Push de tag `vX.Y.Z` | Tests → docker build → push `:latest` + `:vX.Y.Z` → rolling update prod |
| `backend-sqlc.yml` | Toda branch/PR | Valida sqlc, compila, tests |

## Troubleshooting

| Problema | Fix |
|---|---|
| CI `sqlc-build` FAILURE | `make sqlc` + commit los archivos en `db/sqlc/` |
| "Transaction failed" / fila sin ID | Patrón `id=''` — agregar `tools.GenerateNanoid(tx)` antes del `tx.Create()` |
| Cron no dispara en startup | Revisar log: debe aparecer `"cron: first run (post-startup)"` al inicio |
| Forgot password no envía email | Si `RESEND_API_KEY` no está seteado, el email se loguea a stdout — comportamiento esperado en dev |
| `migrate-down` interactivo | Usar `make migrate-down-1` para revertir solo la última; `migrate-down` sin número pide confirmación |
