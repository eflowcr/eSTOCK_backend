-- ============================================================
-- Sprint S3 — SaaS lifecycle DOWN migration
-- Drop in reverse FK dependency order
-- ============================================================

-- Drop indexes added by HR-S3-W1 fixes (before dropping tables)
DROP INDEX IF EXISTS uq_subscriptions_tenant_active;
DROP INDEX IF EXISTS idx_signup_tokens_active_token;

-- Demo data seeds (ref tenants)
DROP TABLE IF EXISTS demo_data_seeds;

-- Signup tokens (no FK to tenants — standalone)
DROP TABLE IF EXISTS signup_tokens;

-- Subscriptions (ref tenants)
DROP TABLE IF EXISTS subscriptions;

-- Tenants (root table — drop last)
DROP TABLE IF EXISTS tenants;
