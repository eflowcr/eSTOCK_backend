-- Presentation conversions: WMS-style conversion factors between presentation types
-- (e.g. 1 Pallet = 20 Cajas, 1 Caja = 12 Unidades). Used for convert mode and logistics.
CREATE TABLE IF NOT EXISTS public.presentation_conversions (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    from_presentation_type_id TEXT NOT NULL REFERENCES public.presentation_types(id) ON DELETE CASCADE,
    to_presentation_type_id TEXT NOT NULL REFERENCES public.presentation_types(id) ON DELETE CASCADE,
    conversion_factor NUMERIC(18, 6) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT presentation_conversions_from_to_unique UNIQUE (from_presentation_type_id, to_presentation_type_id),
    CONSTRAINT presentation_conversions_from_ne_to CHECK (from_presentation_type_id != to_presentation_type_id),
    CONSTRAINT presentation_conversions_factor_positive CHECK (conversion_factor > 0)
);

CREATE INDEX IF NOT EXISTS presentation_conversions_from_idx ON public.presentation_conversions(from_presentation_type_id);
CREATE INDEX IF NOT EXISTS presentation_conversions_to_idx ON public.presentation_conversions(to_presentation_type_id);
CREATE INDEX IF NOT EXISTS presentation_conversions_is_active_idx ON public.presentation_conversions(is_active);

COMMENT ON TABLE public.presentation_conversions IS 'Conversion factors between presentation types: 1 unit of from = conversion_factor units of to (e.g. 1 PALLET = 20 CAJA).';
