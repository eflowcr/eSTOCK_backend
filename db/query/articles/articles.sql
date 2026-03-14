-- Articles CRUD and related queries for sqlc
-- Schema: db/migrations (articles, lots, serials tables)

-- name: ListArticles :many
SELECT id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at
FROM articles
ORDER BY created_at ASC;

-- name: GetArticleByID :one
SELECT id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at
FROM articles
WHERE id = $1
LIMIT 1;

-- name: GetArticleBySku :one
SELECT id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at
FROM articles
WHERE sku = $1
LIMIT 1;

-- name: ArticleExistsBySku :one
SELECT EXISTS(SELECT 1 FROM articles WHERE sku = $1) AS exists;

-- name: CreateArticle :one
INSERT INTO articles (
    sku, name, description, unit_price, presentation,
    track_by_lot, track_by_serial, track_expiration, rotation_strategy,
    min_quantity, max_quantity, image_url
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING id, sku, name, description, unit_price, presentation,
          track_by_lot, track_by_serial, track_expiration, rotation_strategy,
          min_quantity, max_quantity, image_url, is_active,
          created_at, updated_at;

-- name: UpdateArticle :one
UPDATE articles
SET
    sku = $2,
    name = $3,
    description = $4,
    unit_price = $5,
    presentation = $6,
    track_by_lot = $7,
    track_by_serial = $8,
    track_expiration = $9,
    rotation_strategy = $10,
    min_quantity = $11,
    max_quantity = $12,
    image_url = $13,
    is_active = $14,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, sku, name, description, unit_price, presentation,
          track_by_lot, track_by_serial, track_expiration, rotation_strategy,
          min_quantity, max_quantity, image_url, is_active,
          created_at, updated_at;

-- name: DeleteArticle :exec
DELETE FROM articles WHERE id = $1;

-- Lots by SKU (for UpdateArticle warnings)
-- name: ListLotsBySku :many
SELECT id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status
FROM lots
WHERE sku = $1;

-- Serials by SKU (for UpdateArticle warnings)
-- name: ListSerialsBySku :many
SELECT id, serial_number, sku, status, created_at, updated_at
FROM serials
WHERE sku = $1;
