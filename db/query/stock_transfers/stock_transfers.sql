-- Stock transfers and lines. Schema: db/migrations (stock_transfers, stock_transfer_lines).

-- name: ListStockTransfers :many
SELECT id, transfer_number, from_location_id, to_location_id, status, created_by, assigned_to, notes, created_at, updated_at, completed_at
FROM stock_transfers
ORDER BY created_at DESC;

-- name: ListStockTransfersByStatus :many
SELECT id, transfer_number, from_location_id, to_location_id, status, created_by, assigned_to, notes, created_at, updated_at, completed_at
FROM stock_transfers
WHERE status = $1
ORDER BY created_at DESC;

-- name: GetStockTransferByID :one
SELECT id, transfer_number, from_location_id, to_location_id, status, created_by, assigned_to, notes, created_at, updated_at, completed_at
FROM stock_transfers
WHERE id = $1
LIMIT 1;

-- name: GetStockTransferByTransferNumber :one
SELECT id, transfer_number, from_location_id, to_location_id, status, created_by, assigned_to, notes, created_at, updated_at, completed_at
FROM stock_transfers
WHERE transfer_number = $1
LIMIT 1;

-- name: CreateStockTransfer :one
INSERT INTO stock_transfers (transfer_number, from_location_id, to_location_id, status, created_by, assigned_to, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, transfer_number, from_location_id, to_location_id, status, created_by, assigned_to, notes, created_at, updated_at, completed_at;

-- name: UpdateStockTransfer :one
UPDATE stock_transfers
SET from_location_id = $2, to_location_id = $3, status = $4, assigned_to = $5, notes = $6, updated_at = CURRENT_TIMESTAMP,
    completed_at = CASE WHEN $4 = 'completed' AND status != 'completed' THEN CURRENT_TIMESTAMP ELSE completed_at END
WHERE id = $1
RETURNING id, transfer_number, from_location_id, to_location_id, status, created_by, assigned_to, notes, created_at, updated_at, completed_at;

-- name: UpdateStockTransferStatus :one
UPDATE stock_transfers
SET status = $2, updated_at = CURRENT_TIMESTAMP, completed_at = CASE WHEN $2 = 'completed' THEN CURRENT_TIMESTAMP ELSE completed_at END
WHERE id = $1
RETURNING id, transfer_number, from_location_id, to_location_id, status, created_by, assigned_to, notes, created_at, updated_at, completed_at;

-- name: DeleteStockTransfer :exec
DELETE FROM stock_transfers WHERE id = $1;

-- name: ListStockTransferLinesByTransferID :many
SELECT id, stock_transfer_id, sku, quantity, presentation, line_status, created_at
FROM stock_transfer_lines
WHERE stock_transfer_id = $1
ORDER BY created_at ASC;

-- name: GetStockTransferLineByID :one
SELECT id, stock_transfer_id, sku, quantity, presentation, line_status, created_at
FROM stock_transfer_lines
WHERE id = $1
LIMIT 1;

-- name: CreateStockTransferLine :one
INSERT INTO stock_transfer_lines (stock_transfer_id, sku, quantity, presentation, line_status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, stock_transfer_id, sku, quantity, presentation, line_status, created_at;

-- name: UpdateStockTransferLine :one
UPDATE stock_transfer_lines
SET quantity = $2, presentation = $3, line_status = $4
WHERE id = $1
RETURNING id, stock_transfer_id, sku, quantity, presentation, line_status, created_at;

-- name: DeleteStockTransferLine :exec
DELETE FROM stock_transfer_lines WHERE id = $1;

-- name: DeleteStockTransferLinesByTransferID :exec
DELETE FROM stock_transfer_lines WHERE stock_transfer_id = $1;
