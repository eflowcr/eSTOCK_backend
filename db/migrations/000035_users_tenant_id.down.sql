-- 000035_users_tenant_id.down.sql
-- Reverses 000035_users_tenant_id.up.sql.
--
-- Safe in single-tenant environments. In multi-tenant deploys, dropping tenant_id
-- collapses every user into a single namespace; the existing UNIQUE on email keeps
-- referential integrity. Rows themselves are not deleted.

DROP INDEX IF EXISTS idx_users_tenant_email;
ALTER TABLE users DROP COLUMN IF EXISTS tenant_id;
