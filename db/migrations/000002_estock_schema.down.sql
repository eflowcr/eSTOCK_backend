-- =============================================================================
-- eSTOCK schema - ROLLBACK
-- =============================================================================
-- Idempotent: only DROP TABLE IF EXISTS so this is safe when migration 000004
-- down has already dropped these tables (migrate down runs in reverse order).

DROP TABLE IF EXISTS public.picking_tasks;
DROP TABLE IF EXISTS public.inventory_movements;
DROP TABLE IF EXISTS public.receiving_tasks;
DROP TABLE IF EXISTS public.serials;
DROP TABLE IF EXISTS public.inventory_serials;
DROP TABLE IF EXISTS public.inventory_lots;
DROP TABLE IF EXISTS public.lots;
DROP TABLE IF EXISTS public.user_badges;
DROP TABLE IF EXISTS public.badges;
DROP TABLE IF EXISTS public.user_stats;
DROP TABLE IF EXISTS public.stock_alerts;
DROP TABLE IF EXISTS public.adjustments;
DROP TABLE IF EXISTS public.locations;
DROP TABLE IF EXISTS public.inventory;
DROP TABLE IF EXISTS public.articles;
DROP TABLE IF EXISTS public.presentations;
DROP TABLE IF EXISTS public.users;
DROP TABLE IF EXISTS public.sessions;

DROP SEQUENCE IF EXISTS picking_tasks_id_seq;
DROP SEQUENCE IF EXISTS inventory_movements_id_seq;
DROP SEQUENCE IF EXISTS receiving_tasks_id_seq;
DROP SEQUENCE IF EXISTS serials_id_seq;
DROP SEQUENCE IF EXISTS inventory_serials_id_seq;
DROP SEQUENCE IF EXISTS inventory_lots_id_seq;
DROP SEQUENCE IF EXISTS lots_id_seq;
DROP SEQUENCE IF EXISTS user_badges_id_seq;
DROP SEQUENCE IF EXISTS badges_id_seq;
DROP SEQUENCE IF EXISTS user_stats_id_seq;
DROP SEQUENCE IF EXISTS stock_alerts_id_seq;
DROP SEQUENCE IF EXISTS adjustments_id_seq;
DROP SEQUENCE IF EXISTS locations_id_seq;
DROP SEQUENCE IF EXISTS inventory_id_seq;
DROP SEQUENCE IF EXISTS articles_id_seq;
