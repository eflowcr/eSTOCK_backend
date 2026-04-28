-- 000019_tenant_id_operational.up.sql
-- Add tenant_id to operational tables (deferred from S2 M2)
-- Default backfill: existing rows get the global default tenant UUID.
-- Safe because S2 was single-tenant only.

ALTER TABLE picking_tasks
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;
-- TODO S3 (M2 deuda — HR-S2.5-W1): replace with composite index (tenant_id, created_at)
-- to match S2 migration 000018 pattern and eliminate heap sort on list queries.
CREATE INDEX idx_picking_tasks_tenant_id ON picking_tasks(tenant_id);

ALTER TABLE receiving_tasks
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;
-- TODO S3 (M2 deuda): composite (tenant_id, created_at) preferred.
CREATE INDEX idx_receiving_tasks_tenant_id ON receiving_tasks(tenant_id);

ALTER TABLE adjustments
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;
-- TODO S3 (M2 deuda): composite (tenant_id, created_at) preferred.
CREATE INDEX idx_adjustments_tenant_id ON adjustments(tenant_id);

-- Drop the DEFAULT after backfill so future inserts must explicitly set tenant_id.
ALTER TABLE picking_tasks ALTER COLUMN tenant_id DROP DEFAULT;
ALTER TABLE receiving_tasks ALTER COLUMN tenant_id DROP DEFAULT;
ALTER TABLE adjustments ALTER COLUMN tenant_id DROP DEFAULT;

-- Optional: FK to tenants table if it exists (S3 will create it if not).
-- Skipped for now to avoid blocking on tenants table existence.

-- Update inbound_number uniqueness to be per-tenant (HR1 M6).
DROP INDEX IF EXISTS idx_receiving_tasks_inbound_number;
CREATE UNIQUE INDEX idx_receiving_tasks_inbound_number_tenant
  ON receiving_tasks(tenant_id, inbound_number)
  WHERE inbound_number IS NOT NULL;
