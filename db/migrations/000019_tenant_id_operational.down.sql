-- 000019_tenant_id_operational.down.sql
-- Reverses 000019_tenant_id_operational.up.sql

DROP INDEX IF EXISTS idx_receiving_tasks_inbound_number_tenant;
CREATE UNIQUE INDEX idx_receiving_tasks_inbound_number ON receiving_tasks(inbound_number)
  WHERE inbound_number IS NOT NULL;

DROP INDEX IF EXISTS idx_adjustments_tenant_id;
DROP INDEX IF EXISTS idx_receiving_tasks_tenant_id;
DROP INDEX IF EXISTS idx_picking_tasks_tenant_id;

ALTER TABLE adjustments DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE receiving_tasks DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE picking_tasks DROP COLUMN IF EXISTS tenant_id;
