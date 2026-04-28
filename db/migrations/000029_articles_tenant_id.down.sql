-- 000029_articles_tenant_id.down.sql
-- Reverses 000029_articles_tenant_id.up.sql.
--
-- WARNING: this rollback is only safe in single-tenant environments. If multiple
-- tenants share overlapping SKUs (which becomes valid post-up), dropping tenant_id
-- collapses them and the existing `articles_sku_key` global unique remains intact —
-- but the rows themselves stay (no data loss); only the per-tenant scoping disappears.

DROP INDEX IF EXISTS idx_articles_tenant_category;
DROP INDEX IF EXISTS idx_articles_tenant_created;
DROP INDEX IF EXISTS articles_tenant_sku_key;

ALTER TABLE articles DROP COLUMN IF EXISTS tenant_id;
