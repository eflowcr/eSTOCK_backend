-- =============================================================================
-- Drop all tables in public schema (and migrate version table)
-- =============================================================================
-- Run with: psql "$DATABASE_URL" -f db/scripts/drop_all_tables.sql
-- Or: make db-drop-all
-- After this, run "make migrate-up" to reapply migrations.
-- =============================================================================

-- Drop migrate version table so migrate tool sees a clean slate
DROP TABLE IF EXISTS public.schema_migrations;

-- Drop all tables in public (CASCADE drops dependent objects)
DROP TABLE IF EXISTS public.audit_logs CASCADE;
DROP TABLE IF EXISTS public.user_preferences CASCADE;
DROP TABLE IF EXISTS public.location_types CASCADE;
DROP TABLE IF EXISTS public.inventory_movements CASCADE;
DROP TABLE IF EXISTS public.picking_tasks CASCADE;
DROP TABLE IF EXISTS public.receiving_tasks CASCADE;
DROP TABLE IF EXISTS public.inventory_serials CASCADE;
DROP TABLE IF EXISTS public.inventory_lots CASCADE;
DROP TABLE IF EXISTS public.serials CASCADE;
DROP TABLE IF EXISTS public.lots CASCADE;
DROP TABLE IF EXISTS public.user_badges CASCADE;
DROP TABLE IF EXISTS public.badges CASCADE;
DROP TABLE IF EXISTS public.user_stats CASCADE;
DROP TABLE IF EXISTS public.stock_alerts CASCADE;
DROP TABLE IF EXISTS public.locations CASCADE;
DROP TABLE IF EXISTS public.adjustments CASCADE;
DROP TABLE IF EXISTS public.inventory CASCADE;
DROP TABLE IF EXISTS public.articles CASCADE;
DROP TABLE IF EXISTS public.presentations CASCADE;
DROP TABLE IF EXISTS public.sessions CASCADE;
DROP TABLE IF EXISTS public.users CASCADE;
DROP TABLE IF EXISTS public.session_types CASCADE;
DROP TABLE IF EXISTS public.roles CASCADE;

-- Drop helper functions
DROP FUNCTION IF EXISTS public.set_default_user_role();
DROP FUNCTION IF EXISTS public.set_updated_by_from_session();
