-- 000034_inventory_lots_tenant_isolation.up.sql
-- S3.5 W2-A: Multi-tenant isolation for inventory_lots junction table.
--
-- Problem (HR-S3-W5 / S3.5 audit): inventory_lots is a junction table linking
-- inventory ↔ lots with per-location quantity. It carries no tenant_id, so once
-- a second tenant is provisioned, lots aggregation queries (GROUP BY location)
-- and inventory-by-location queries can mix data across tenants. The parent
-- inventory and lots tables themselves are tracked under separate W2 waves
-- (W2-B handles lots); this migration adds isolation at the junction level so
-- writes from controllers can be tenant-scoped going forward.
--
-- Fix:
--   1. Add tenant_id with default backfill to the existing default tenant.
--   2. Add composite UNIQUE (tenant_id, inventory_id, lot_id, location) to
--      prevent duplicate (per-tenant) allocations of the same lot at the same
--      location to the same inventory row.
--   3. Drop column DEFAULT after backfill so future inserts must set
--      tenant_id explicitly.
--   4. Composite indexes for the two common query shapes:
--        a. WHERE tenant_id = ? AND lot_id = ?  (lot trace by tenant)
--        b. WHERE tenant_id = ? AND inventory_id = ?  (lots-for-inventory)

ALTER TABLE inventory_lots
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;

CREATE UNIQUE INDEX uq_inventory_lots_tenant_inv_lot_loc
  ON inventory_lots(tenant_id, inventory_id, lot_id, location);

CREATE INDEX idx_inventory_lots_tenant_lot
  ON inventory_lots(tenant_id, lot_id);

CREATE INDEX idx_inventory_lots_tenant_inventory
  ON inventory_lots(tenant_id, inventory_id);

ALTER TABLE inventory_lots ALTER COLUMN tenant_id DROP DEFAULT;
