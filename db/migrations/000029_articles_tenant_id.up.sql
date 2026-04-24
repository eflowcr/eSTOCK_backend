-- 000029_articles_tenant_id.up.sql
-- Sprint S3.5 W1 — articles tenant isolation (HR-S3-W5 C2 fix).
--
-- Articles table was born tenant-agnostic (000002_estock_schema). With S3 SaaS
-- multi-tenant signup live, every second tenant signing up was silently inheriting
-- tenant 1's articles via SeedFarma's `WHERE sku=? FirstOrCreate` pattern.
--
-- This migration adds tenant_id, backfills with the default tenant UUID
-- (matching the convention in 000019_tenant_id_operational and 000023_saas_lifecycle),
-- and adds a per-tenant composite unique on (tenant_id, sku).
--
-- FK NOTE: We intentionally KEEP the existing `articles_sku_key` global UNIQUE on
-- (sku) for now. There are 8+ FKs referencing articles(sku) (inventory.sku,
-- stock_transfer_items.sku, article_suppliers.article_sku, purchase_order_items.article_sku,
-- sales_order_items.article_sku, delivery_note_items.article_sku, backorders.article_sku,
-- and similar). Dropping the global unique would require redesigning every dependent
-- FK as composite (tenant_id, sku). That cross-table retrofit is deferred to a future
-- sprint after every dependent table has its own tenant_id (W2 covers lots/inventory_lots/
-- locations/serials, but inventory and stock_transfer_items still need work). The new
-- composite unique below is what the application code uses for per-tenant SKU dedup;
-- the global unique remains as a safety net + FK target until then.

-- 1. Add tenant_id column with default backfill to default tenant (matches S2/S3 convention).
ALTER TABLE articles
  ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'::uuid;

-- 2. Drop the DEFAULT after backfill so future inserts must explicitly set tenant_id.
ALTER TABLE articles ALTER COLUMN tenant_id DROP DEFAULT;

-- 3. Per-tenant SKU uniqueness — application reads dedup via this index.
--
-- KNOWN LIMITATION: until the FK retrofit lands, the legacy `articles_sku_key`
-- (global UNIQUE on sku) still prevents two different tenants from registering
-- the same SKU value. The composite below is what application logic reads from
-- (per-tenant lookups, "exists by tenant + sku" checks, etc.); it becomes the
-- ENFORCEMENT mechanism only once the global unique is dropped — see
-- feedback_estock_articles_no_tenant_isolation.md and the W2/W4 follow-ups.
--
-- For S3.5 W1 specifically, this means SeedFarma for a SECOND tenant must use
-- SKUs that don't collide with the first tenant's catalog. The W4 wave will
-- update SeedFarma to either prefix SKUs per-tenant or share the master catalog
-- explicitly. Fixing the data-leak symptom (tenant 2 inheriting tenant 1 rows
-- silently) is achieved here by scoping every read/write by tenant_id; the
-- "same SKU across tenants" capability is a separate, larger structural change.
CREATE UNIQUE INDEX articles_tenant_sku_key ON articles(tenant_id, sku);

-- 4. Composite covering index for the common list query (sorted by created_at DESC).
CREATE INDEX idx_articles_tenant_created ON articles(tenant_id, created_at DESC);

-- 5. Composite index for category filter (common in catalog UI). category_id was added in S2 M2.
CREATE INDEX idx_articles_tenant_category ON articles(tenant_id, category_id)
  WHERE category_id IS NOT NULL;

-- 6. Optional: FK to tenants table for referential integrity (HR-S2.5 deferred from 000019).
--    Skipped to keep migration reversible and consistent with 000019; can be added later.
