-- Restore code column for rollback
ALTER TABLE public.roles ADD COLUMN IF NOT EXISTS code TEXT;
UPDATE public.roles SET code = LOWER(name) WHERE code IS NULL OR code = '';
ALTER TABLE public.roles ALTER COLUMN code SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS roles_code_key ON public.roles (code);

CREATE OR REPLACE FUNCTION public.set_default_user_role()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.role_id IS NULL OR NEW.role_id = '' THEN
        SELECT id INTO NEW.role_id FROM public.roles WHERE code = 'operator' LIMIT 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE public.roles IS 'RBAC roles; code is stable (admin, operator, viewer).';
