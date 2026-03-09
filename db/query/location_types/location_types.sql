-- Location types CRUD for sqlc. Schema: db/migrations (location_types table)

-- name: ListLocationTypes :many
SELECT id, code, name, sort_order, is_active, created_at, updated_at
FROM location_types
WHERE is_active = true
ORDER BY sort_order ASC, code ASC;

-- name: ListLocationTypesAdmin :many
SELECT id, code, name, sort_order, is_active, created_at, updated_at
FROM location_types
ORDER BY sort_order ASC, code ASC;

-- name: GetLocationTypeByID :one
SELECT id, code, name, sort_order, is_active, created_at, updated_at
FROM location_types
WHERE id = $1
LIMIT 1;

-- name: GetLocationTypeByCode :one
SELECT id, code, name, sort_order, is_active, created_at, updated_at
FROM location_types
WHERE code = $1
LIMIT 1;

-- name: LocationTypeExistsByCode :one
SELECT EXISTS(SELECT 1 FROM location_types WHERE code = $1) AS exists;

-- name: CreateLocationType :one
INSERT INTO location_types (code, name, sort_order, is_active)
VALUES ($1, $2, $3, $4)
RETURNING id, code, name, sort_order, is_active, created_at, updated_at;

-- name: UpdateLocationType :one
UPDATE location_types
SET
    code = $2,
    name = $3,
    sort_order = $4,
    is_active = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, code, name, sort_order, is_active, created_at, updated_at;

-- name: DeleteLocationType :exec
DELETE FROM location_types WHERE id = $1;
