-- Revert M4: restore the original composite unique constraint and drop partial indexes.
DROP INDEX IF EXISTS uq_categories_child_name;
DROP INDEX IF EXISTS uq_categories_root_name;
ALTER TABLE categories ADD CONSTRAINT categories_tenant_id_name_parent_id_key UNIQUE (tenant_id, name, parent_id);

-- Revert M8 auxiliary index.
DROP INDEX IF EXISTS idx_clients_tenant_type_active;
