-- Categories CRUD for sqlc
-- Schema: db/migrations/000018_sprint_s2.up.sql (categories table)

-- name: CreateCategory :one
INSERT INTO categories (id, tenant_id, name, parent_id, is_active)
VALUES ($1, $2, $3, $4, true)
RETURNING id, tenant_id, name, parent_id, is_active, created_at, updated_at;

-- name: GetCategoryByID :one
SELECT id, tenant_id, name, parent_id, is_active, created_at, updated_at
FROM categories WHERE id = $1;

-- name: ListCategoriesByTenant :many
SELECT id, tenant_id, name, parent_id, is_active, created_at, updated_at
FROM categories WHERE tenant_id = $1 ORDER BY name ASC;

-- name: UpdateCategory :one
UPDATE categories
SET name = $2, parent_id = $3, updated_at = now()
WHERE id = $1
RETURNING id, tenant_id, name, parent_id, is_active, created_at, updated_at;

-- name: SoftDeleteCategory :exec
UPDATE categories SET is_active = false, updated_at = now() WHERE id = $1;
