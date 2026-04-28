-- 000033_serials_tenant_isolation.up.sql
-- S3.5 W2-A: Multi-tenant isolation for serials master-data table.
--
-- Problem (HR-S3-W5 / S3.5 audit): serials has no tenant_id and no UNIQUE on
-- serial_number. While there is no global UNIQUE to drop, queries by SKU or by
-- ID return rows from any tenant, so tenant 2's signup would expose tenant 1's
-- serials. Adding a composite UNIQUE (tenant_id, serial_number) also prevents
-- silent cross-tenant duplicates.
--
-- Fix:
--   1. Add tenant_id with default backfill to the existing default tenant.
--   2. Add composite UNIQUE (tenant_id, serial_number) — no pre-existing
--      global UNIQUE to drop. FKs (sales_order_items.serial_id) reference id,
--      not serial_number, so unaffected.
--   3. Drop the column DEFAULT after backfill.
--   4. Composite (tenant_id, sku) index supports the common
--      "list serials by sku for tenant" filter.

ALTER TABLE serials
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;

CREATE UNIQUE INDEX uq_serials_tenant_serial_number
  ON serials(tenant_id, serial_number);

CREATE INDEX idx_serials_tenant_sku
  ON serials(tenant_id, sku);

ALTER TABLE serials ALTER COLUMN tenant_id DROP DEFAULT;
