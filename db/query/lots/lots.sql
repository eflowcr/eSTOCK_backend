-- Lots CRUD for sqlc
-- Schema: db/migrations (lots table)

-- name: ListLots :many
SELECT id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
       lot_notes, manufactured_at, best_before_date
FROM lots
ORDER BY created_at DESC;

-- name: GetLotByID :one
SELECT id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
       lot_notes, manufactured_at, best_before_date
FROM lots
WHERE id = $1
LIMIT 1;

-- name: CreateLot :one
INSERT INTO lots (lot_number, sku, quantity, expiration_date, status,
                 lot_notes, manufactured_at, best_before_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
          lot_notes, manufactured_at, best_before_date;

-- name: UpdateLot :one
UPDATE lots
SET
    lot_number = $2,
    sku = $3,
    quantity = $4,
    expiration_date = $5,
    status = $6,
    lot_notes = $7,
    manufactured_at = $8,
    best_before_date = $9,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
          lot_notes, manufactured_at, best_before_date;

-- name: DeleteLot :exec
DELETE FROM lots WHERE id = $1;
