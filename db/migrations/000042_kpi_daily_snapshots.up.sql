-- ============================================================
-- KPI daily snapshots — powers the dashboard sparklines (web only).
-- One row per (tenant, day) with the 3 headline KPIs. The cron
-- RunKpiSnapshot upserts today's row daily; this migration backfills
-- ~6 monthly points per tenant from existing data so the sparklines
-- show a curve from day one (demo-friendly). Mobile never reads this.
-- Migration 000042 (2026-05-29).
-- ============================================================

CREATE TABLE IF NOT EXISTS kpi_daily_snapshots (
  id              TEXT PRIMARY KEY DEFAULT nanoid(),
  tenant_id       UUID NOT NULL,
  snapshot_date   DATE NOT NULL,
  total_skus      INT NOT NULL DEFAULT 0,
  active_tasks    INT NOT NULL DEFAULT 0,
  low_stock_count INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_kpi_snap_tenant_date
  ON kpi_daily_snapshots (tenant_id, snapshot_date DESC);

-- ── Backfill: 6 monthly points (last 6 month-ends incl. current) per tenant ──
-- total_skus: cumulative count of active articles created on/before the month-end.
-- active_tasks: cumulative count of picking+receiving tasks created on/before the month-end
--   (arranque aproximado; el cron real luego guarda el conteo exacto de open+in_progress).
-- low_stock_count: count of distinct SKUs whose latest movement on/before the month-end
--   left remaining_stock < 20 (best historical signal available from existing data).
WITH months AS (
  SELECT (date_trunc('month', CURRENT_DATE) - (g || ' months')::interval)::date
         + interval '1 month' - interval '1 day' AS month_end
  FROM generate_series(0, 5) AS g
),
tenants_active AS (
  SELECT DISTINCT tenant_id FROM articles WHERE tenant_id IS NOT NULL
)
INSERT INTO kpi_daily_snapshots (tenant_id, snapshot_date, total_skus, active_tasks, low_stock_count)
SELECT
  t.tenant_id,
  m.month_end::date,
  COALESCE((
    SELECT COUNT(*) FROM articles a
    WHERE a.tenant_id = t.tenant_id
      AND a.is_active IS NOT FALSE
      AND a.created_at <= m.month_end
  ), 0) AS total_skus,
  COALESCE((
    SELECT COUNT(*) FROM (
      SELECT created_at FROM receiving_tasks WHERE created_at <= m.month_end
      UNION ALL
      SELECT created_at FROM picking_tasks WHERE created_at <= m.month_end
    ) tk
  ), 0) AS active_tasks,
  COALESCE((
    SELECT COUNT(DISTINCT im.sku) FROM inventory_movements im
    WHERE im.created_at <= m.month_end
      AND im.remaining_stock IS NOT NULL
      AND im.remaining_stock < 20
  ), 0) AS low_stock_count
FROM tenants_active t
CROSS JOIN months m
ON CONFLICT (tenant_id, snapshot_date) DO NOTHING;
