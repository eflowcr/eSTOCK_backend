-- 000031_stock_alerts_tenant_isolation.down.sql
-- Reverses 000031_stock_alerts_tenant_isolation.up.sql.

DROP INDEX IF EXISTS idx_stock_alerts_tenant_resolved_created_at;

ALTER TABLE stock_alerts DROP COLUMN IF EXISTS tenant_id;
