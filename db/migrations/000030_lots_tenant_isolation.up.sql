-- 000030_lots_tenant_isolation.up.sql
-- S3.5 W2-B: Add tenant_id to lots table to close multi-tenant master-data gap.
--
-- Background: lots had no tenant_id, so two tenants with the same lot_number on the
-- same SKU would collide and queries would return cross-tenant rows. Articles and
-- inventory tables already (or will, in S3.5 W1/W2-A) carry tenant_id; lots was the
-- last operational dependency missing isolation for the receiving/picking flows.
--
-- Backfill strategy: existing lots belong to the single live tenant (G2). Default
-- to the global default tenant UUID, then DROP DEFAULT so future inserts must set
-- tenant_id explicitly (matches pattern from 000019).

ALTER TABLE lots
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;

-- Composite index: tenant filter + most common sort key (created_at DESC for ListLots).
CREATE INDEX IF NOT EXISTS idx_lots_tenant_created_at ON lots(tenant_id, created_at DESC);

-- Composite index for the SKU lookup pattern (rotation/picking).
CREATE INDEX IF NOT EXISTS idx_lots_tenant_sku ON lots(tenant_id, sku);

-- Per-tenant uniqueness on (lot_number, sku). Two tenants may legitimately use the
-- same lot_number/SKU pair, but within one tenant lot_number must be unique per SKU
-- so receiving/picking can resolve a lot deterministically.
CREATE UNIQUE INDEX IF NOT EXISTS uq_lots_tenant_sku_lot_number
  ON lots(tenant_id, sku, lot_number);

ALTER TABLE lots ALTER COLUMN tenant_id DROP DEFAULT;
