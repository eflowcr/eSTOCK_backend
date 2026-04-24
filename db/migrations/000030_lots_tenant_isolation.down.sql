-- 000030_lots_tenant_isolation.down.sql
-- Reverses 000030_lots_tenant_isolation.up.sql.
--
-- WARNING: dropping tenant_id collapses isolation. Safe only on single-tenant
-- prod (G2 as of S3.5). If multiple tenants share data and this is rolled back,
-- the unique (sku, lot_number) constraint that previously existed implicitly
-- via application logic will no longer be enforced — lot lookups may return
-- rows from other tenants.

DROP INDEX IF EXISTS uq_lots_tenant_sku_lot_number;
DROP INDEX IF EXISTS idx_lots_tenant_sku;
DROP INDEX IF EXISTS idx_lots_tenant_created_at;

ALTER TABLE lots DROP COLUMN IF EXISTS tenant_id;
