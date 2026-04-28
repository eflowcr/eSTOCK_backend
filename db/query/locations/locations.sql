-- Locations CRUD for sqlc
-- Schema: db/migrations (locations table; tenant_id added in 000032).
-- All HTTP-facing endpoints MUST filter by tenant_id (S3.5 W2-A).

-- name: ListLocationsByTenant :many
-- S3.5 W2-A: tenant_id guard prevents cross-tenant location enumeration.
SELECT id, location_code, description, zone, type, is_active, is_way_out, created_at, updated_at, tenant_id
FROM locations
WHERE tenant_id = $1
ORDER BY created_at ASC;

-- name: GetLocationByIDForTenant :one
-- S3.5 W2-A: tenant_id guard prevents cross-tenant id lookup.
SELECT id, location_code, description, zone, type, is_active, is_way_out, created_at, updated_at, tenant_id
FROM locations
WHERE id = $1 AND tenant_id = $2
LIMIT 1;

-- name: GetLocationByLocationCodeForTenant :one
-- S3.5 W2-A: tenant_id guard. Used as fallback by ID lookup when caller passed a code.
SELECT id, location_code, description, zone, type, is_active, is_way_out, created_at, updated_at, tenant_id
FROM locations
WHERE location_code = $1 AND tenant_id = $2
LIMIT 1;

-- name: LocationExistsByLocationCodeForTenant :one
-- S3.5 W2-A: tenant_id guard. Used by Create to enforce per-tenant unique location_code.
SELECT EXISTS(
  SELECT 1 FROM locations WHERE location_code = $1 AND tenant_id = $2
) AS exists;

-- name: CreateLocation :one
-- S3.5 W2-A: tenant_id is required and provided by the controller layer.
INSERT INTO locations (location_code, description, zone, type, is_active, is_way_out, tenant_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, location_code, description, zone, type, is_active, is_way_out, created_at, updated_at, tenant_id;

-- name: UpdateLocationForTenant :one
-- S3.5 W2-A: tenant_id guard prevents cross-tenant update.
UPDATE locations
SET
    location_code = $2,
    description = $3,
    zone = $4,
    type = $5,
    is_active = $6,
    is_way_out = $7,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND tenant_id = $8
RETURNING id, location_code, description, zone, type, is_active, is_way_out, created_at, updated_at, tenant_id;

-- name: DeleteLocationForTenant :exec
-- S3.5 W2-A: tenant_id guard prevents cross-tenant delete.
DELETE FROM locations WHERE id = $1 AND tenant_id = $2;
