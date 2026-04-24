-- 000032_locations_tenant_isolation.down.sql
-- Reverse W2-A locations tenant isolation.

DROP INDEX IF EXISTS idx_locations_tenant_created_at;
DROP INDEX IF EXISTS uq_locations_tenant_code;

-- Recreate the original global unique on location_code.
CREATE UNIQUE INDEX IF NOT EXISTS locations_location_code_key
  ON locations (location_code);

ALTER TABLE locations DROP COLUMN IF EXISTS tenant_id;
