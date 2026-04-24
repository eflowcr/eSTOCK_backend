-- 000033_serials_tenant_isolation.down.sql
-- Reverse W2-A serials tenant isolation.

DROP INDEX IF EXISTS idx_serials_tenant_sku;
DROP INDEX IF EXISTS uq_serials_tenant_serial_number;

ALTER TABLE serials DROP COLUMN IF EXISTS tenant_id;
