-- Locations CRUD for sqlc
-- Schema: db/migrations (locations table)

-- name: ListLocations :many
SELECT id, location_code, description, zone, type, is_active, created_at, updated_at
FROM locations
ORDER BY created_at ASC;

-- name: GetLocationByID :one
SELECT id, location_code, description, zone, type, is_active, created_at, updated_at
FROM locations
WHERE id = $1
LIMIT 1;

-- name: GetLocationByLocationCode :one
SELECT id, location_code, description, zone, type, is_active, created_at, updated_at
FROM locations
WHERE location_code = $1
LIMIT 1;

-- name: LocationExistsByLocationCode :one
SELECT EXISTS(SELECT 1 FROM locations WHERE location_code = $1) AS exists;

-- name: CreateLocation :one
INSERT INTO locations (location_code, description, zone, type, is_active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, location_code, description, zone, type, is_active, created_at, updated_at;

-- name: UpdateLocation :one
UPDATE locations
SET
    location_code = $2,
    description = $3,
    zone = $4,
    type = $5,
    is_active = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, location_code, description, zone, type, is_active, created_at, updated_at;

-- name: DeleteLocation :exec
DELETE FROM locations WHERE id = $1;
