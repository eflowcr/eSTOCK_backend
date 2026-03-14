-- Stock transfers: WMS-style transfer orders (from location → to location) with lines (SKU, quantity).
-- Schema depends on: locations (id), articles (sku).

CREATE TABLE IF NOT EXISTS public.stock_transfers (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    transfer_number VARCHAR(50) NOT NULL,
    from_location_id TEXT NOT NULL REFERENCES public.locations(id),
    to_location_id TEXT NOT NULL REFERENCES public.locations(id),
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_by TEXT NOT NULL,
    assigned_to TEXT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    CONSTRAINT chk_stock_transfer_from_to_different CHECK (from_location_id != to_location_id),
    CONSTRAINT chk_stock_transfer_status CHECK (status IN ('draft', 'in_progress', 'completed', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS stock_transfers_transfer_number_key ON public.stock_transfers (transfer_number);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_from_location ON public.stock_transfers (from_location_id);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_to_location ON public.stock_transfers (to_location_id);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_status ON public.stock_transfers (status);
CREATE INDEX IF NOT EXISTS idx_stock_transfers_created_at ON public.stock_transfers (created_at DESC);

CREATE TABLE IF NOT EXISTS public.stock_transfer_lines (
    id TEXT PRIMARY KEY DEFAULT nanoid(),
    stock_transfer_id TEXT NOT NULL REFERENCES public.stock_transfers(id) ON DELETE CASCADE,
    sku VARCHAR NOT NULL REFERENCES public.articles(sku),
    quantity NUMERIC(10,3) NOT NULL,
    presentation VARCHAR,
    line_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_stock_transfer_line_quantity CHECK (quantity > 0),
    CONSTRAINT chk_stock_transfer_line_status CHECK (line_status IN ('pending', 'picked', 'received', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_stock_transfer_lines_transfer_id ON public.stock_transfer_lines (stock_transfer_id);
CREATE INDEX IF NOT EXISTS idx_stock_transfer_lines_sku ON public.stock_transfer_lines (sku);
