-- Articles CRUD and related queries for sqlc.
-- Schema: db/migrations (articles, lots, serials tables).
-- S3.5 W1 — every read/write is now tenant-scoped via tenant_id (HR-S3-W5 C2).
-- The legacy global queries (ListArticles, GetArticleBySku, ArticleExistsBySku) remain
-- ONLY for internal lookups (FK validation, stock alerts cron, dashboards) where the
-- caller has no JWT/Config tenant context. HTTP endpoints MUST use the *ForTenant variants.

-- name: ListArticles :many
-- INTERNAL USE ONLY (cron, FK lookups). HTTP handlers must call ListArticlesForTenant.
SELECT id, tenant_id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at,
       category_id, shelf_life_in_days, safety_stock, batch_number_series,
       serial_number_series, min_order_qty, default_location_id,
       receiving_notes, shipping_notes
FROM articles
ORDER BY created_at ASC;

-- name: ListArticlesForTenant :many
-- HTTP-facing list. Uses idx_articles_tenant_created (composite covering index).
SELECT id, tenant_id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at,
       category_id, shelf_life_in_days, safety_stock, batch_number_series,
       serial_number_series, min_order_qty, default_location_id,
       receiving_notes, shipping_notes
FROM articles
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: GetArticleByID :one
-- INTERNAL USE ONLY. HTTP handlers must call GetArticleByIDForTenant.
SELECT id, tenant_id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at,
       category_id, shelf_life_in_days, safety_stock, batch_number_series,
       serial_number_series, min_order_qty, default_location_id,
       receiving_notes, shipping_notes
FROM articles
WHERE id = $1
LIMIT 1;

-- name: GetArticleByIDForTenant :one
-- HR-style tenant guard. Use for HTTP responses to prevent cross-tenant enumeration.
SELECT id, tenant_id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at,
       category_id, shelf_life_in_days, safety_stock, batch_number_series,
       serial_number_series, min_order_qty, default_location_id,
       receiving_notes, shipping_notes
FROM articles
WHERE id = $1 AND tenant_id = $2
LIMIT 1;

-- name: GetArticleBySku :one
-- INTERNAL USE ONLY (FK lookups, dashboards). HTTP handlers must call GetArticleBySkuForTenant.
SELECT id, tenant_id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at,
       category_id, shelf_life_in_days, safety_stock, batch_number_series,
       serial_number_series, min_order_qty, default_location_id,
       receiving_notes, shipping_notes
FROM articles
WHERE sku = $1
LIMIT 1;

-- name: GetArticleBySkuForTenant :one
-- Per-tenant SKU lookup. Hits articles_tenant_sku_key index.
SELECT id, tenant_id, sku, name, description, unit_price, presentation,
       track_by_lot, track_by_serial, track_expiration, rotation_strategy,
       min_quantity, max_quantity, image_url, is_active,
       created_at, updated_at,
       category_id, shelf_life_in_days, safety_stock, batch_number_series,
       serial_number_series, min_order_qty, default_location_id,
       receiving_notes, shipping_notes
FROM articles
WHERE sku = $1 AND tenant_id = $2
LIMIT 1;

-- name: ArticleExistsBySku :one
-- INTERNAL USE ONLY. Tenant-scoped variant below.
SELECT EXISTS(SELECT 1 FROM articles WHERE sku = $1) AS exists;

-- name: ArticleExistsBySkuForTenant :one
SELECT EXISTS(SELECT 1 FROM articles WHERE sku = $1 AND tenant_id = $2) AS exists;

-- name: CreateArticle :one
-- All inserts now require tenant_id ($1).
INSERT INTO articles (
    tenant_id, sku, name, description, unit_price, presentation,
    track_by_lot, track_by_serial, track_expiration, rotation_strategy,
    min_quantity, max_quantity, image_url,
    category_id, shelf_life_in_days, safety_stock, batch_number_series,
    serial_number_series, min_order_qty, default_location_id,
    receiving_notes, shipping_notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17, $18, $19, $20, $21, $22
)
RETURNING id, tenant_id, sku, name, description, unit_price, presentation,
          track_by_lot, track_by_serial, track_expiration, rotation_strategy,
          min_quantity, max_quantity, image_url, is_active,
          created_at, updated_at,
          category_id, shelf_life_in_days, safety_stock, batch_number_series,
          serial_number_series, min_order_qty, default_location_id,
          receiving_notes, shipping_notes;

-- name: UpdateArticle :one
-- Tenant guard via WHERE id = $1 AND tenant_id = $24 — prevents cross-tenant update.
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
    category_id = $15,
    shelf_life_in_days = $16,
    safety_stock = $17,
    batch_number_series = $18,
    serial_number_series = $19,
    min_order_qty = $20,
    default_location_id = $21,
    receiving_notes = $22,
    shipping_notes = $23,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND tenant_id = $24
RETURNING id, tenant_id, sku, name, description, unit_price, presentation,
          track_by_lot, track_by_serial, track_expiration, rotation_strategy,
          min_quantity, max_quantity, image_url, is_active,
          created_at, updated_at,
          category_id, shelf_life_in_days, safety_stock, batch_number_series,
          serial_number_series, min_order_qty, default_location_id,
          receiving_notes, shipping_notes;

-- name: DeleteArticle :exec
-- Tenant guard prevents cross-tenant delete.
DELETE FROM articles WHERE id = $1 AND tenant_id = $2;

-- Lots by SKU (for UpdateArticle warnings) — internal, no tenant filter (lots table
-- not yet tenant-scoped; tracked in S3.5 W2).
-- name: ListLotsBySku :many
SELECT id, lot_number, sku, quantity, expiration_date, created_at, updated_at, status,
       lot_notes, manufactured_at, best_before_date
FROM lots
WHERE sku = $1;

-- Serials by SKU (for UpdateArticle warnings) — internal, no tenant filter (serials
-- table not yet tenant-scoped; tracked in S3.5 W2).
-- name: ListSerialsBySku :many
SELECT id, serial_number, sku, status, created_at, updated_at
FROM serials
WHERE sku = $1;
