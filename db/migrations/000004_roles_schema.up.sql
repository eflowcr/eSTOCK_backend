-- =============================================================================
-- Roles table for RBAC: id (role name), permissions JSONB, metadata
-- users.role (varchar) references this id; no FK to avoid migration of existing data.
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    permissions JSONB NOT NULL DEFAULT '{}',
    is_active bool NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE roles IS 'RBAC roles; users.role stores role id (e.g. admin, operator, viewer)';

-- Seed default roles (id = role name for simple lookup from users.role)
INSERT INTO public.roles (id, name, description, permissions) VALUES
    ('admin', 'Admin', 'Full access', '{"all": true}'),
    ('operator', 'Operator', 'Create, read, update for articles, lots, locations, serials', '{"articles": {"create": true, "read": true, "update": true, "delete": true}, "lots": {"create": true, "read": true, "update": true, "delete": true}, "locations": {"create": true, "read": true, "update": true, "delete": true}, "serials": {"create": true, "read": true, "update": true, "delete": true}, "inventory": {"create": true, "read": true, "update": true, "delete": true}}'),
    ('viewer', 'Viewer', 'Read-only', '{"articles": {"read": true}, "lots": {"read": true}, "locations": {"read": true}, "serials": {"read": true}, "inventory": {"read": true}}')
ON CONFLICT (id) DO NOTHING;
