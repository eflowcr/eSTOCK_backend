-- Presentation conversions CRUD. Schema: db/migrations (presentation_conversions table)

-- name: ListPresentationConversions :many
SELECT pc.id, pc.from_presentation_type_id, pc.to_presentation_type_id, pc.conversion_factor, pc.is_active, pc.created_at, pc.updated_at
FROM presentation_conversions pc
WHERE pc.is_active = true
ORDER BY pc.from_presentation_type_id, pc.to_presentation_type_id;

-- name: ListPresentationConversionsAdmin :many
SELECT pc.id, pc.from_presentation_type_id, pc.to_presentation_type_id, pc.conversion_factor, pc.is_active, pc.created_at, pc.updated_at
FROM presentation_conversions pc
ORDER BY pc.from_presentation_type_id, pc.to_presentation_type_id;

-- name: GetPresentationConversionByID :one
SELECT id, from_presentation_type_id, to_presentation_type_id, conversion_factor, is_active, created_at, updated_at
FROM presentation_conversions
WHERE id = $1
LIMIT 1;

-- name: GetPresentationConversionByFromAndTo :one
SELECT id, from_presentation_type_id, to_presentation_type_id, conversion_factor, is_active, created_at, updated_at
FROM presentation_conversions
WHERE from_presentation_type_id = $1 AND to_presentation_type_id = $2
LIMIT 1;

-- name: CreatePresentationConversion :one
INSERT INTO presentation_conversions (from_presentation_type_id, to_presentation_type_id, conversion_factor, is_active)
VALUES ($1, $2, $3, $4)
RETURNING id, from_presentation_type_id, to_presentation_type_id, conversion_factor, is_active, created_at, updated_at;

-- name: UpdatePresentationConversion :one
UPDATE presentation_conversions
SET
    conversion_factor = $2,
    is_active = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, from_presentation_type_id, to_presentation_type_id, conversion_factor, is_active, created_at, updated_at;

-- name: DeletePresentationConversion :exec
DELETE FROM presentation_conversions WHERE id = $1;
