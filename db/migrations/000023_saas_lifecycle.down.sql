-- ============================================================
-- Sprint S3 — SaaS lifecycle DOWN migration
-- Drop in reverse FK dependency order
-- ============================================================

-- Demo data seeds (ref tenants)
DROP TABLE IF EXISTS demo_data_seeds;

-- Signup tokens (no FK to tenants — standalone)
DROP TABLE IF EXISTS signup_tokens;

-- Subscriptions (ref tenants)
DROP TABLE IF EXISTS subscriptions;

-- Tenants (root table — drop last)
DROP TABLE IF EXISTS tenants;
