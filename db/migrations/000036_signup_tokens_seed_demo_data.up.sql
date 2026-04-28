-- Migration 000036: Add seed_demo_data flag to signup_tokens (S3.7 companion).
--
-- Sprint S3.7 W3 (frontend) shipped a "Cargar datos de demostración" checkbox
-- on the signup form (PR #32). The frontend now sends `seed_demo_data` in the
-- POST /api/signup payload, but until this migration the backend ignored it
-- and ALWAYS ran SeedFarma — every brand-new tenant got 100 SKUs / 20 receiving
-- tasks / 15 picking tasks / 10 locations whether they wanted them or not.
--
-- This column persists the user's choice on the pending signup_tokens row so
-- VerifySignup can honor it when finalising the tenant.
--
-- Default TRUE preserves backwards compatibility:
--   * Pre-existing rows (still pending verification) keep seeding as before.
--   * Old frontend builds that don't send the field default to seeding too.
--   * Only an explicit `seed_demo_data: false` from the new frontend opts out.

ALTER TABLE signup_tokens
  ADD COLUMN IF NOT EXISTS seed_demo_data BOOLEAN NOT NULL DEFAULT TRUE;
