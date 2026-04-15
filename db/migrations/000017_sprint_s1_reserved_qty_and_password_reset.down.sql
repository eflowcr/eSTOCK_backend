DROP INDEX IF EXISTS idx_lots_sku_lot_number_active_unique;
ALTER TABLE public.inventory DROP CONSTRAINT IF EXISTS chk_reserved_non_negative;
ALTER TABLE public.inventory DROP COLUMN IF EXISTS reserved_qty;
DROP TABLE IF EXISTS public.password_reset_tokens;
