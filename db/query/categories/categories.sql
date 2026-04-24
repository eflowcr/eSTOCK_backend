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
-- M8: Push search/is_active filters and pagination to SQL (HR1 deferred).
-- Pass NULL for any optional param to skip that filter.
SELECT id, tenant_id, name, parent_id, is_active, created_at, updated_at
FROM categories
WHERE tenant_id = $1
  AND ($2::boolean IS NULL OR is_active = $2)
  -- TODO(S3/MA4): escape ILIKE wildcards with replace($3,'%','\%') ESCAPE '\' to prevent
  -- DoS via catastrophic backtracking on strings containing many '%' or '_' characters.
  AND ($3::text IS NULL OR name ILIKE '%' || $3 || '%')
ORDER BY name ASC
LIMIT COALESCE($4::int, 200)
OFFSET COALESCE($5::int, 0);

-- name: UpdateCategory :one
UPDATE categories
SET name = $2, parent_id = $3, updated_at = now()
WHERE id = $1
RETURNING id, tenant_id, name, parent_id, is_active, created_at, updated_at;

-- name: SoftDeleteCategory :exec
UPDATE categories SET is_active = false, updated_at = now() WHERE id = $1;
