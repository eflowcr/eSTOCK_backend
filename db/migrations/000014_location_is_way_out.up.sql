-- Location way out / dock: mark locations used as dock, loading bay, or exit point (WMS).
ALTER TABLE public.locations
ADD COLUMN IF NOT EXISTS is_way_out BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN public.locations.is_way_out IS 'When true, location is used as dock, loading bay, or exit point in WMS.';
