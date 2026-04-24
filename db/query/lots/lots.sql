-- Lots CRUD for sqlc
-- Schema: db/migrations (lots table; tenant_id added in 000030)
-- S3.5 W2-B: every public query is tenant-scoped. Internal helpers (GetLotByID) keep
-- the un-scoped variant for cross-domain joins (lot trace, picking task validation)
-- where the caller already proved tenancy via the parent record.
--
-- Column order in SELECT/RETURNING must match db/sqlc/models.go::Lot (Postgres appends
-- tenant_id at the end after the 000030 migration), otherwise sqlc generates per-query
-- Row structs instead of returning the shared Lot model.

-- name: ListLots :many
-- Returns all lots for a tenant, sorted by created_at DESC.
SELECT id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
       lot_notes, manufactured_at, best_before_date, tenant_id
FROM lots
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: ListLotsBySkuForTenant :many
-- Tenant-scoped lookup by SKU; replaces the un-scoped ListLotsBySku for HTTP callers.
SELECT id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
       lot_notes, manufactured_at, best_before_date, tenant_id
FROM lots
WHERE tenant_id = $1 AND sku = $2
ORDER BY created_at DESC;

-- name: GetLotByID :one
-- Internal use only: no tenant filter. Use GetLotByIDForTenant for HTTP callers.
SELECT id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
       lot_notes, manufactured_at, best_before_date, tenant_id
FROM lots
WHERE id = $1
LIMIT 1;

-- name: GetLotByIDForTenant :one
-- Tenant guard prevents cross-tenant lot enumeration via HTTP (S3.5 W2-B).
SELECT id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
       lot_notes, manufactured_at, best_before_date, tenant_id
FROM lots
WHERE id = $1 AND tenant_id = $2
LIMIT 1;

-- name: CreateLot :one
INSERT INTO lots (lot_number, sku, quantity, expiration_date, status,
                 lot_notes, manufactured_at, best_before_date, tenant_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
          lot_notes, manufactured_at, best_before_date, tenant_id;

-- name: UpdateLot :one
-- Tenant guard prevents cross-tenant updates (S3.5 W2-B).
UPDATE lots
SET
    lot_number = $3,
    sku = $4,
    quantity = $5,
    expiration_date = $6,
    status = $7,
    lot_notes = $8,
    manufactured_at = $9,
    best_before_date = $10,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND tenant_id = $2
RETURNING id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
          lot_notes, manufactured_at, best_before_date, tenant_id;

-- name: DeleteLot :exec
-- Tenant guard prevents cross-tenant deletes (S3.5 W2-B).
DELETE FROM lots WHERE id = $1 AND tenant_id = $2;
