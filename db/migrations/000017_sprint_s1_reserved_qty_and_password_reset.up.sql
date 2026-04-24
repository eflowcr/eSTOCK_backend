-- ============================================================
-- Sprint S1: reserved_qty en inventory + password reset tokens
--            + unicidad de lote por SKU (A4)
-- ============================================================

-- 1. Agregar reserved_qty a inventory
ALTER TABLE public.inventory
  ADD COLUMN IF NOT EXISTS reserved_qty NUMERIC(10,3) NOT NULL DEFAULT 0;

-- Solo validar que no sea negativo — el límite superior se maneja en app code.
-- Un constraint CHECK (reserved_qty <= quantity) causaría errores en transacciones
-- donde se actualiza quantity y reserved_qty en pasos separados dentro del mismo tx.
ALTER TABLE public.inventory
  ADD CONSTRAINT chk_reserved_non_negative
    CHECK (reserved_qty >= 0);

-- 2. Tabla para tokens de reset de contraseña
CREATE TABLE IF NOT EXISTS public.password_reset_tokens (
  id           VARCHAR NOT NULL,
  user_id      VARCHAR NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  token        VARCHAR NOT NULL UNIQUE,
  expires_at   TIMESTAMPTZ NOT NULL,
  used_at      TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON public.password_reset_tokens(token);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON public.password_reset_tokens(user_id);

-- 3. A4: Unicidad de lote por SKU (lot_number + sku únicos para lotes no archivados)
-- Permite re-usar un lot_number si el lote anterior con ese número fue archivado
-- (ej: lote completado y cerrado, luego un nuevo pedido con mismo nombre).
-- Requiere que la columna lots.status exista (ya existe, default 'pending').
CREATE UNIQUE INDEX IF NOT EXISTS idx_lots_sku_lot_number_active_unique
  ON public.lots(sku, lot_number)
  WHERE status IS NULL OR status != 'archived';
