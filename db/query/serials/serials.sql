-- Serials CRUD for sqlc
-- Schema: db/migrations (serials table; tenant_id added in 000033).
-- All HTTP-facing endpoints MUST filter by tenant_id (S3.5 W2-A).
--
-- NOTE: ListSerialsBySku (no tenant filter) is defined in db/query/articles/articles.sql
-- and used internally by article-update warning logic. That query is NOT for HTTP responses.
-- Use ListSerialsBySkuForTenant for tenant-scoped HTTP endpoints.

-- name: GetSerialByIDForTenant :one
-- S3.5 W2-A: tenant_id guard prevents cross-tenant id lookup.
SELECT id, serial_number, sku, status, created_at, updated_at, tenant_id
FROM serials
WHERE id = $1 AND tenant_id = $2
LIMIT 1;

-- name: ListSerialsBySkuForTenant :many
-- S3.5 W2-A: tenant_id guard. Replaces global ListSerialsBySku for HTTP responses.
SELECT id, serial_number, sku, status, created_at, updated_at, tenant_id
FROM serials
WHERE sku = $1 AND tenant_id = $2
ORDER BY created_at DESC;

-- name: CreateSerial :one
-- S3.5 W2-A: tenant_id is required and provided by the controller layer.
INSERT INTO serials (serial_number, sku, status, tenant_id)
VALUES ($1, $2, $3, $4)
RETURNING id, serial_number, sku, status, created_at, updated_at, tenant_id;

-- name: UpdateSerialForTenant :one
-- S3.5 W2-A: tenant_id guard prevents cross-tenant update.
UPDATE serials
SET
    serial_number = $2,
    sku = $3,
    status = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND tenant_id = $5
RETURNING id, serial_number, sku, status, created_at, updated_at, tenant_id;

-- name: DeleteSerialForTenant :exec
-- S3.5 W2-A: tenant_id guard prevents cross-tenant delete.
DELETE FROM serials WHERE id = $1 AND tenant_id = $2;
