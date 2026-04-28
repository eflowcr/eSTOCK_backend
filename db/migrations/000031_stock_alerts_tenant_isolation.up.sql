-- 000031_stock_alerts_tenant_isolation.up.sql
-- S3.5 W2-B: Add tenant_id to stock_alerts table.
--
-- Background: stock_alerts.Analyze() recomputed alerts globally (TRUNCATE +
-- recompute over all inventory rows). With S2.5 inventory now tenant-scoped,
-- Analyze() must run per tenant and the resulting alert rows must carry
-- tenant_id so the dashboard, exports and notifications stay isolated.
--
-- Backfill: existing rows belong to the single live tenant (G2). Default to the
-- global default tenant UUID, then DROP DEFAULT so future inserts must set
-- tenant_id explicitly.

ALTER TABLE stock_alerts
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;

-- Composite index: tenant filter + the GetAllStockAlerts/Summary filter (is_resolved)
-- and sort key (created_at). Covers both list endpoints in one index.
CREATE INDEX IF NOT EXISTS idx_stock_alerts_tenant_resolved_created_at
  ON stock_alerts(tenant_id, is_resolved, created_at);

ALTER TABLE stock_alerts ALTER COLUMN tenant_id DROP DEFAULT;
