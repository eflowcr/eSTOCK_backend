-- Clients CRUD for sqlc
-- Schema: db/migrations/000018_sprint_s2.up.sql (clients table)

-- name: CreateClient :one
INSERT INTO clients (id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, $11)
RETURNING id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at;

-- name: GetClientByID :one
-- Internal use only: no tenant filter. Use GetClientByIDForTenant for HTTP responses (HR1-M3).
SELECT id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at
FROM clients WHERE id = $1;

-- name: GetClientByIDForTenant :one
-- HR1-M3: tenant_id guard prevents cross-tenant client enumeration via HTTP.
SELECT id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at
FROM clients WHERE id = $1 AND tenant_id = $2;

-- name: GetClientByTenantAndCode :one
SELECT id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at
FROM clients WHERE tenant_id = $1 AND code = $2;

-- name: ListClientsByTenant :many
SELECT id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at
FROM clients WHERE tenant_id = $1 ORDER BY name ASC;

-- name: UpdateClient :one
-- HR1-M3: tenant_id guard prevents cross-tenant update.
UPDATE clients
SET type = $2, code = $3, name = $4, email = $5, phone = $6, address = $7, tax_id = $8, notes = $9, updated_at = now()
WHERE id = $1 AND tenant_id = $10
RETURNING id, tenant_id, type, code, name, email, phone, address, tax_id, notes, is_active, created_by, created_at, updated_at;

-- name: SoftDeleteClient :exec
-- HR1-M3: tenant_id guard prevents cross-tenant soft-delete.
UPDATE clients SET is_active = false, updated_at = now() WHERE id = $1 AND tenant_id = $2;
