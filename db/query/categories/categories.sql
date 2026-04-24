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
-- sqlc.narg() used so generated struct has named fields (IsActive, Search, Limit, Offset)
-- instead of positional Column2…Column5 names that sqlc v1.29.0 infers for $N::type IS NULL patterns.
SELECT id, tenant_id, name, parent_id, is_active, created_at, updated_at
FROM categories
WHERE tenant_id = $1
  AND (sqlc.narg('is_active')::boolean IS NULL OR is_active = sqlc.narg('is_active'))
  -- TODO(S3/MA4): escape ILIKE wildcards with replace(search,'%','\%') ESCAPE '\' to prevent
  -- DoS via catastrophic backtracking on strings containing many '%' or '_' characters.
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY name ASC
LIMIT COALESCE(sqlc.narg('limit')::int, 200)
OFFSET COALESCE(sqlc.narg('offset')::int, 0);

-- name: UpdateCategory :one
UPDATE categories
SET name = $2, parent_id = $3, updated_at = now()
WHERE id = $1
RETURNING id, tenant_id, name, parent_id, is_active, created_at, updated_at;

-- name: SoftDeleteCategory :exec
UPDATE categories SET is_active = false, updated_at = now() WHERE id = $1;
