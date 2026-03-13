-- Presentation types: configurable types for article/inventory presentation (e.g. Unidad, Caja, Pallet). Admin-managed.
CREATE TABLE IF NOT EXISTS public.presentation_types (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS presentation_types_code_idx ON public.presentation_types(code);
CREATE INDEX IF NOT EXISTS presentation_types_is_active_idx ON public.presentation_types(is_active);
CREATE INDEX IF NOT EXISTS presentation_types_sort_order_idx ON public.presentation_types(sort_order);

COMMENT ON TABLE public.presentation_types IS 'Presentation type catalog (Unidad, Caja, Pallet, Paquete, etc.); used by articles.presentation as code reference';

-- Seed default types (Unidad, Caja, Pallet, Paquete)
INSERT INTO public.presentation_types (id, code, name, sort_order, is_active) VALUES
    (nanoid(), 'UNIDAD', 'Unidad', 10, true),
    (nanoid(), 'CAJA', 'Caja', 20, true),
    (nanoid(), 'PALLET', 'Pallet', 30, true),
    (nanoid(), 'PAQUETE', 'Paquete', 40, true)
ON CONFLICT (code) DO NOTHING;
