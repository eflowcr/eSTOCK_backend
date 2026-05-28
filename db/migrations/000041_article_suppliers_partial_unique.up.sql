-- ============================================================
-- S3-5 close — partial UNIQUE on article_suppliers so a soft-deleted
-- link can be recreated. The original table-level
-- UNIQUE(tenant_id, article_sku, supplier_id) didn't differentiate
-- soft-deleted rows → re-adding a previously-removed supplier failed
-- with 23505. Replace it with a partial UNIQUE index that only
-- enforces uniqueness among rows where deleted_at IS NULL.
-- Migration 000041 (2026-05-28).
-- ============================================================

ALTER TABLE article_suppliers
  DROP CONSTRAINT IF EXISTS article_suppliers_tenant_id_article_sku_supplier_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS article_suppliers_active_unique
  ON article_suppliers (tenant_id, article_sku, supplier_id)
  WHERE deleted_at IS NULL;
