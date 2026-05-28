DROP INDEX IF EXISTS article_suppliers_active_unique;

ALTER TABLE article_suppliers
  ADD CONSTRAINT article_suppliers_tenant_id_article_sku_supplier_id_key
  UNIQUE (tenant_id, article_sku, supplier_id);
