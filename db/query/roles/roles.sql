-- Roles for RBAC: get by id or name (case-insensitive), get permissions only

-- name: GetRoleByID :one
SELECT id, name, description, permissions, is_active, created_at, updated_at
FROM roles
WHERE id = $1 OR LOWER(name) = LOWER($1)
LIMIT 1;

-- name: GetRolePermissions :one
SELECT permissions
FROM roles
WHERE (id = $1 OR LOWER(name) = LOWER($1)) AND is_active = true
LIMIT 1;

-- name: ListRoles :many
SELECT id, name, description, permissions, is_active, created_at, updated_at
FROM roles
ORDER BY name;

-- name: UpdateRolePermissions :one
UPDATE roles
SET permissions = $2, updated_at = now()
WHERE id = $1
RETURNING id, name, description, permissions, is_active, created_at, updated_at;
