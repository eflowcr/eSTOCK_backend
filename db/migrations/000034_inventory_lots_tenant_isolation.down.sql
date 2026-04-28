-- 000034_inventory_lots_tenant_isolation.down.sql
-- Reverse W2-A inventory_lots tenant isolation.

DROP INDEX IF EXISTS idx_inventory_lots_tenant_inventory;
DROP INDEX IF EXISTS idx_inventory_lots_tenant_lot;
DROP INDEX IF EXISTS uq_inventory_lots_tenant_inv_lot_loc;

ALTER TABLE inventory_lots DROP COLUMN IF EXISTS tenant_id;
