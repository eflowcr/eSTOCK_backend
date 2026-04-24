-- 000032_locations_tenant_isolation.up.sql
-- S3.5 W2-A: Multi-tenant isolation for locations master-data table.
--
-- Problem (HR-S3-W5 / S3.5 audit): locations.location_code carries a GLOBAL
-- UNIQUE index (locations_location_code_key), so tenant 2 cannot create any
-- location whose code already exists for tenant 1. Lists/lookups also leak
-- across tenants because no WHERE tenant_id = ? clause exists.
--
-- Fix:
--   1. Add tenant_id with default backfill to the existing default tenant
--      (matches pattern from 000019_tenant_id_operational.up.sql).
--   2. Drop the global UNIQUE on location_code; replace with composite
--      UNIQUE (tenant_id, location_code). FKs on locations.id (e.g.
--      stock_transfers.from_location_id, articles.default_location_id) are
--      unaffected — they reference id, not location_code.
--   3. Drop the column DEFAULT after backfill so future inserts must set
--      tenant_id explicitly.
--   4. Composite (tenant_id, created_at) index for ORDER BY in list queries.

ALTER TABLE locations
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;

-- Drop the GLOBAL unique index on location_code (was: locations_location_code_key).
DROP INDEX IF EXISTS locations_location_code_key;

-- Per-tenant uniqueness on location_code.
CREATE UNIQUE INDEX uq_locations_tenant_code
  ON locations(tenant_id, location_code);

-- Index supporting "list locations for tenant ordered by created_at".
CREATE INDEX idx_locations_tenant_created_at
  ON locations(tenant_id, created_at);

-- Strip default so future inserts must set tenant_id explicitly.
ALTER TABLE locations ALTER COLUMN tenant_id DROP DEFAULT;
