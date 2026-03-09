-- User preferences table (from backend_template). One row per user; GET creates defaults if missing.
CREATE TABLE IF NOT EXISTS public.user_preferences (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    user_id TEXT NOT NULL UNIQUE REFERENCES public.users(id) ON DELETE CASCADE,
    theme TEXT NOT NULL DEFAULT 'system' CHECK (theme IN ('light', 'dark', 'system')),
    language TEXT NOT NULL DEFAULT 'en' CHECK (language IN ('en', 'es', 'fr', 'de')),
    email_notifications BOOLEAN NOT NULL DEFAULT true,
    push_notifications BOOLEAN NOT NULL DEFAULT false,
    marketing_notifications BOOLEAN NOT NULL DEFAULT false,
    profile_visibility TEXT NOT NULL DEFAULT 'private' CHECK (profile_visibility IN ('public', 'private', 'friends')),
    data_sharing BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON public.user_preferences(user_id);

CREATE OR REPLACE FUNCTION public.update_user_preferences_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_user_preferences_updated_at
    BEFORE UPDATE ON public.user_preferences
    FOR EACH ROW
    EXECUTE PROCEDURE public.update_user_preferences_updated_at();

COMMENT ON TABLE public.user_preferences IS 'Per-user preferences (theme, language, notifications, privacy).';
