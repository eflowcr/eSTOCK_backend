-- Location types: configurable types for locations (e.g. Pallet, Shelf, Bin). Admin-managed.
CREATE TABLE IF NOT EXISTS public.location_types (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS location_types_code_idx ON public.location_types(code);
CREATE INDEX IF NOT EXISTS location_types_is_active_idx ON public.location_types(is_active);
CREATE INDEX IF NOT EXISTS location_types_sort_order_idx ON public.location_types(sort_order);

COMMENT ON TABLE public.location_types IS 'Location type catalog (Pallet, Shelf, Bin, etc.); used by locations.type as code reference';

-- Seed default types (match current frontend: PALLET, SHELF, BIN, FLOOR, BLOCK)
INSERT INTO public.location_types (id, code, name, sort_order, is_active) VALUES
    (nanoid(), 'PALLET', 'Pallet', 10, true),
    (nanoid(), 'SHELF', 'Shelf', 20, true),
    (nanoid(), 'BIN', 'Bin', 30, true),
    (nanoid(), 'FLOOR', 'Floor', 40, true),
    (nanoid(), 'BLOCK', 'Block', 50, true)
ON CONFLICT (code) DO NOTHING;
