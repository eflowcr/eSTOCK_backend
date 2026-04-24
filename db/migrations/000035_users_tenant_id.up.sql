-- 000035_users_tenant_id.up.sql
-- Sprint S3.5 W5.5 — HR-S3.5 C2 fix: add tenant_id to users table.
--
-- Background: through S3.5 W3 the JWT carries a tenant_id claim, but Login at
-- repositories/authentication_repository.go:66 stamped Config.TenantID (the env-injected
-- pod default) into every token because users had no tenant_id of their own. Result: a
-- user belonging to tenant 2 logging in via /api/auth/login received a JWT claiming
-- tenant 1, and from then on read/wrote tenant 1's data — exactly the cross-tenant leak
-- S3.5 set out to close.
--
-- This migration adds users.tenant_id, backfills existing rows with the default tenant
-- UUID '00000000-0000-0000-0000-000000000001' (consistent with 000019/000023/000029-34),
-- drops the DEFAULT so future inserts must set tenant_id explicitly, and adds a composite
-- index on (tenant_id, email) to keep login lookups fast.
--
-- Production note: the live deployment is single-tenant, so the DEFAULT backfill is
-- correct (every existing user belongs to tenant 1). For new users created via
-- /api/signup (S3.5 W4), repositories/signup_repository.go:VerifySignup stamps the
-- freshly-created tenant's UUID into users.tenant_id explicitly.

-- 1. Add tenant_id column with default backfill (matches S2/S3 pattern).
ALTER TABLE users
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;

-- 2. Drop the DEFAULT after backfill so every future INSERT must set tenant_id.
ALTER TABLE users ALTER COLUMN tenant_id DROP DEFAULT;

-- 3. Composite index for login lookup (WHERE email = ?).
--    The existing UNIQUE on email (cross-tenant) remains as a safety net until a
--    later sprint redesigns email uniqueness as per-tenant. For now login can scope
--    by (tenant_id, email) when subdomain routing is wired; until then the index
--    just speeds up the email lookup + makes a future per-tenant unique cheap to add.
CREATE INDEX IF NOT EXISTS idx_users_tenant_email
  ON users(tenant_id, email);
