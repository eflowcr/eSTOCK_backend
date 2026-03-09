DROP TRIGGER IF EXISTS update_user_preferences_updated_at ON public.user_preferences;
DROP FUNCTION IF EXISTS public.update_user_preferences_updated_at();
DROP INDEX IF EXISTS idx_user_preferences_user_id;
DROP TABLE IF EXISTS public.user_preferences;
