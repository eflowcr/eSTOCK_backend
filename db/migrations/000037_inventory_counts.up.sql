-- Inventory counts: cycle counts / physical inventory sheets.
-- Pattern reference: ERPNext stock_reconciliation + Dolibarr inventory module.
-- Workflow: draft → scheduled → in_progress → in_review → submitted (creates adjustment) | cancelled
-- Schema depends on: locations(id), users(id), adjustments(id) (created in 000004 with TEXT id).

CREATE TABLE IF NOT EXISTS public.inventory_counts (
    id            TEXT PRIMARY KEY DEFAULT nanoid(),
    code          VARCHAR(50) NOT NULL,
    name          VARCHAR(200) NOT NULL,
    description   TEXT,
    status        VARCHAR(20) NOT NULL DEFAULT 'draft',
    scheduled_for TIMESTAMP,
    started_at    TIMESTAMP,
    completed_at  TIMESTAMP,
    submitted_at  TIMESTAMP,
    submitted_by  TEXT REFERENCES public.users(id),
    adjustment_id TEXT REFERENCES public.adjustments(id),
    created_by    TEXT NOT NULL REFERENCES public.users(id),
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_inventory_counts_status CHECK (status IN ('draft', 'scheduled', 'in_progress', 'in_review', 'submitted', 'cancelled')),
    CONSTRAINT uniq_inventory_counts_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_inventory_counts_status ON public.inventory_counts(status);
CREATE INDEX IF NOT EXISTS idx_inventory_counts_created_at ON public.inventory_counts(created_at DESC);

CREATE TABLE IF NOT EXISTS public.inventory_count_locations (
    id          TEXT PRIMARY KEY DEFAULT nanoid(),
    count_id    TEXT NOT NULL REFERENCES public.inventory_counts(id) ON DELETE CASCADE,
    location_id TEXT NOT NULL REFERENCES public.locations(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    assigned_to TEXT REFERENCES public.users(id),
    CONSTRAINT chk_inventory_count_locations_status CHECK (status IN ('pending', 'in_progress', 'done')),
    CONSTRAINT uniq_inventory_count_locations UNIQUE (count_id, location_id)
);

CREATE INDEX IF NOT EXISTS idx_inventory_count_locations_count ON public.inventory_count_locations(count_id);

CREATE TABLE IF NOT EXISTS public.inventory_count_lines (
    id           TEXT PRIMARY KEY DEFAULT nanoid(),
    count_id     TEXT NOT NULL REFERENCES public.inventory_counts(id) ON DELETE CASCADE,
    location_id  TEXT NOT NULL REFERENCES public.locations(id),
    sku          VARCHAR(100) NOT NULL,
    lot          VARCHAR(100),
    serial       VARCHAR(100),
    expected_qty NUMERIC(12,4) NOT NULL DEFAULT 0,
    scanned_qty  NUMERIC(12,4) NOT NULL,
    variance_qty NUMERIC(12,4) NOT NULL,
    note         TEXT,
    scanned_by   TEXT NOT NULL REFERENCES public.users(id),
    scanned_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inventory_count_lines_count ON public.inventory_count_lines(count_id);
CREATE INDEX IF NOT EXISTS idx_inventory_count_lines_sku ON public.inventory_count_lines(sku);
CREATE INDEX IF NOT EXISTS idx_inventory_count_lines_location ON public.inventory_count_lines(location_id);

COMMENT ON TABLE public.inventory_counts IS 'Cycle/physical count sheets (mobile-driven). variance_qty on each line drives auto-adjustment on submit.';
COMMENT ON COLUMN public.inventory_count_lines.variance_qty IS 'scanned_qty - expected_qty; positive = stock found extra, negative = stock missing.';
