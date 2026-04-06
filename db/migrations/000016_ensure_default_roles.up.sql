-- Ensure the three default roles always exist.
-- Safe to run on any instance (new or existing). Uses WHERE NOT EXISTS to avoid duplicates.

INSERT INTO public.roles (name, description, permissions)
SELECT 'Admin', 'Full access', '{"all": true}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM public.roles WHERE LOWER(name) = 'admin');

INSERT INTO public.roles (name, description, permissions)
SELECT 'Operator', 'Create, read, update for articles, lots, locations, serials',
    '{"lots": {"read": true, "create": true, "delete": true, "update": true}, "serials": {"read": true, "create": true, "delete": true, "update": true}, "articles": {"read": true, "create": true, "delete": true, "update": true}, "inventory": {"read": true, "create": true, "delete": true, "update": true}, "locations": {"read": true, "create": true, "delete": true, "update": true}}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM public.roles WHERE LOWER(name) = 'operator');

INSERT INTO public.roles (name, description, permissions)
SELECT 'Viewer', 'Read-only access to all resources',
    '{"lots": {"read": true}, "serials": {"read": true}, "articles": {"read": true}, "inventory": {"read": true}, "locations": {"read": true}}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM public.roles WHERE LOWER(name) = 'viewer');
