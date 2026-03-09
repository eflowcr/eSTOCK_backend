-- Drop code column from roles; name is the valid identifier for lookups.
ALTER TABLE public.roles DROP COLUMN IF EXISTS code;
DROP INDEX IF EXISTS public.roles_code_key;

-- Update default-role trigger to use name (case-insensitive)
CREATE OR REPLACE FUNCTION public.set_default_user_role()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.role_id IS NULL OR NEW.role_id = '' THEN
        SELECT id INTO NEW.role_id FROM public.roles WHERE LOWER(name) = 'operator' LIMIT 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON TABLE public.roles IS 'RBAC roles; name is the stable identifier (Admin, Operator, Viewer).';
