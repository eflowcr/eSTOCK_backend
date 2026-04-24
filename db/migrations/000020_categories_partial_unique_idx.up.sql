-- M4: Replace single composite unique constraint with two partial indexes so that
-- root categories (parent_id IS NULL) and child categories (parent_id IS NOT NULL)
-- are scoped correctly.  The old UNIQUE(tenant_id, name, parent_id) treated NULLs as
-- distinct, meaning two root categories with the same name could coexist (NULL != NULL).
-- The two partial indexes below enforce uniqueness correctly for each case.

ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_tenant_id_name_parent_id_key;

CREATE UNIQUE INDEX uq_categories_root_name
  ON categories(tenant_id, name)
  WHERE parent_id IS NULL;

CREATE UNIQUE INDEX uq_categories_child_name
  ON categories(tenant_id, name, parent_id)
  WHERE parent_id IS NOT NULL;

-- M8 auxiliary: composite covering index for the common (tenant_id, type, is_active) filter
-- in ListClientsByTenant SQL queries.  IF NOT EXISTS is safe for idempotency.
CREATE INDEX IF NOT EXISTS idx_clients_tenant_type_active
  ON clients(tenant_id, type, is_active);
