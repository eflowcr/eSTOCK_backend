-- Clients CRUD for sqlc
-- Schema: db/migrations/000018_sprint_s2.up.sql (clients table)

-- name: CreateClient :one
INSERT INTO clients (id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, $11)
RETURNING id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at;

-- name: GetClientByID :one
SELECT id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at
FROM clients WHERE id = $1;

-- name: GetClientByTenantAndCode :one
SELECT id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at
FROM clients WHERE tenant_id = $1 AND code = $2;

-- name: ListClientsByTenant :many
SELECT id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at
FROM clients WHERE tenant_id = $1 ORDER BY name ASC;

-- name: UpdateClient :one
UPDATE clients
SET type = $2, code = $3, name = $4, email = $5, phone = $6, address = $7, tax_id = $8, notes = $9, updated_at = now()
WHERE id = $1
RETURNING id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at;

-- name: SoftDeleteClient :exec
UPDATE clients SET is_active = false, updated_at = now() WHERE id = $1;
