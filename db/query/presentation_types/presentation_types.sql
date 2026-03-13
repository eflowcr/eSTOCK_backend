-- Presentation types CRUD for sqlc. Schema: db/migrations (presentation_types table)

-- name: ListPresentationTypes :many
SELECT id, code, name, sort_order, is_active, created_at, updated_at
FROM presentation_types
WHERE is_active = true
ORDER BY sort_order ASC, code ASC;

-- name: ListPresentationTypesAdmin :many
SELECT id, code, name, sort_order, is_active, created_at, updated_at
FROM presentation_types
ORDER BY sort_order ASC, code ASC;

-- name: GetPresentationTypeByID :one
SELECT id, code, name, sort_order, is_active, created_at, updated_at
FROM presentation_types
WHERE id = $1
LIMIT 1;

-- name: GetPresentationTypeByCode :one
SELECT id, code, name, sort_order, is_active, created_at, updated_at
FROM presentation_types
WHERE code = $1
LIMIT 1;

-- name: PresentationTypeExistsByCode :one
SELECT EXISTS(SELECT 1 FROM presentation_types WHERE code = $1) AS exists;

-- name: CreatePresentationType :one
INSERT INTO presentation_types (code, name, sort_order, is_active)
VALUES ($1, $2, $3, $4)
RETURNING id, code, name, sort_order, is_active, created_at, updated_at;

-- name: UpdatePresentationType :one
UPDATE presentation_types
SET
    code = $2,
    name = $3,
    sort_order = $4,
    is_active = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, code, name, sort_order, is_active, created_at, updated_at;

-- name: DeletePresentationType :exec
DELETE FROM presentation_types WHERE id = $1;
