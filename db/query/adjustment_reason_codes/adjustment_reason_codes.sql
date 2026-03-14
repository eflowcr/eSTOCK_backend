-- Adjustment reason codes CRUD for sqlc. Schema: db/migrations (adjustment_reason_codes table)

-- name: ListAdjustmentReasonCodes :many
SELECT id, code, name, direction, is_system, display_order, is_active, created_at, updated_at
FROM adjustment_reason_codes
WHERE is_active = true
ORDER BY display_order ASC, code ASC;

-- name: ListAdjustmentReasonCodesAdmin :many
SELECT id, code, name, direction, is_system, display_order, is_active, created_at, updated_at
FROM adjustment_reason_codes
ORDER BY display_order ASC, code ASC;

-- name: GetAdjustmentReasonCodeByID :one
SELECT id, code, name, direction, is_system, display_order, is_active, created_at, updated_at
FROM adjustment_reason_codes
WHERE id = $1
LIMIT 1;

-- name: GetAdjustmentReasonCodeByCode :one
SELECT id, code, name, direction, is_system, display_order, is_active, created_at, updated_at
FROM adjustment_reason_codes
WHERE code = $1
LIMIT 1;

-- name: CreateAdjustmentReasonCode :one
INSERT INTO adjustment_reason_codes (code, name, direction, is_system, display_order, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, name, direction, is_system, display_order, is_active, created_at, updated_at;

-- name: UpdateAdjustmentReasonCode :one
UPDATE adjustment_reason_codes
SET
    code = $2,
    name = $3,
    direction = $4,
    display_order = $5,
    is_active = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, code, name, direction, is_system, display_order, is_active, created_at, updated_at;

-- name: DeleteAdjustmentReasonCode :exec
DELETE FROM adjustment_reason_codes WHERE id = $1 AND is_system = false;
