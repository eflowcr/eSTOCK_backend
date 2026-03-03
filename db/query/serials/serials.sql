-- Serials CRUD for sqlc
-- Schema: db/migrations (serials table)

-- name: GetSerialByID :one
SELECT id, serial_number, sku, status, created_at, updated_at
FROM serials
WHERE id = $1
LIMIT 1;

-- name: CreateSerial :one
INSERT INTO serials (serial_number, sku, status)
VALUES ($1, $2, $3)
RETURNING id, serial_number, sku, status, created_at, updated_at;

-- name: UpdateSerial :one
UPDATE serials
SET
    serial_number = $2,
    sku = $3,
    status = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, serial_number, sku, status, created_at, updated_at;

-- name: DeleteSerial :exec
DELETE FROM serials WHERE id = $1;
