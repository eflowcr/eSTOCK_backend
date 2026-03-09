-- =============================================================================
-- 000004 down — Drop roles, users, sessions and dependent schema (reverse order)
-- =============================================================================
-- Leaves DB without eSTOCK app tables (only 000001 nanoid remains). Re-run
-- 000001–000003 and 000004 to restore.
-- =============================================================================

ALTER TABLE IF EXISTS public.audit_logs DROP CONSTRAINT IF EXISTS audit_logs_user_id_fkey;
ALTER TABLE IF EXISTS public.inventory DROP CONSTRAINT IF EXISTS inventory_sku_fkey;
ALTER TABLE IF EXISTS public.user_badges DROP CONSTRAINT IF EXISTS user_badges_badge_id_fkey;
ALTER TABLE IF EXISTS public.inventory_lots DROP CONSTRAINT IF EXISTS inventory_lots_inventory_id_fkey;
ALTER TABLE IF EXISTS public.inventory_lots DROP CONSTRAINT IF EXISTS inventory_lots_lot_id_fkey;
ALTER TABLE IF EXISTS public.inventory_serials DROP CONSTRAINT IF EXISTS inventory_serials_inventory_id_fkey;
ALTER TABLE IF EXISTS public.inventory_serials DROP CONSTRAINT IF EXISTS inventory_serials_serial_id_fkey;
ALTER TABLE IF EXISTS public.users DROP CONSTRAINT IF EXISTS users_role_id_fkey;
ALTER TABLE IF EXISTS public.sessions DROP CONSTRAINT IF EXISTS sessions_user_id_fkey;
ALTER TABLE IF EXISTS public.sessions DROP CONSTRAINT IF EXISTS sessions_session_type_id_fkey;

DROP TABLE IF EXISTS public.audit_logs;
DROP TABLE IF EXISTS public.inventory_movements;
DROP TABLE IF EXISTS public.picking_tasks;
DROP TABLE IF EXISTS public.receiving_tasks;
DROP TABLE IF EXISTS public.inventory_serials;
DROP TABLE IF EXISTS public.inventory_lots;
DROP TABLE IF EXISTS public.serials;
DROP TABLE IF EXISTS public.lots;
DROP TABLE IF EXISTS public.user_badges;
DROP TABLE IF EXISTS public.badges;
DROP TABLE IF EXISTS public.user_stats;
DROP TABLE IF EXISTS public.stock_alerts;
DROP TABLE IF EXISTS public.locations;
DROP TABLE IF EXISTS public.adjustments;
DROP TABLE IF EXISTS public.inventory;
DROP TABLE IF EXISTS public.articles;
DROP TABLE IF EXISTS public.presentations;
DROP TABLE IF EXISTS public.sessions;
DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS public.session_types;
DROP TABLE IF EXISTS public.roles;

DROP FUNCTION IF EXISTS public.set_default_user_role();
DROP FUNCTION IF EXISTS public.set_updated_by_from_session();
