-- =============================================================================
-- Performance indexes: batch queries, alert filtering, dashboard counts
-- =============================================================================

-- inventory: SKU lookups (N+1 killer in GetAllInventory and stock alert analysis)
CREATE INDEX IF NOT EXISTS idx_inventory_sku ON inventory(sku);

-- stock_alerts: is_resolved filter (GetAllStockAlerts, Summary — called on every page load)
CREATE INDEX IF NOT EXISTS idx_stock_alerts_is_resolved ON stock_alerts(is_resolved);
CREATE INDEX IF NOT EXISTS idx_stock_alerts_sku ON stock_alerts(sku);

-- inventory_movements: analyze() batch query filters on (movement_type, created_at)
CREATE INDEX IF NOT EXISTS idx_inventory_movements_type_created
    ON inventory_movements(movement_type, created_at DESC);

-- lots: expiration alert query + SKU lookups
CREATE INDEX IF NOT EXISTS idx_lots_sku ON lots(sku);
CREATE INDEX IF NOT EXISTS idx_lots_expiration_date ON lots(expiration_date DESC);

-- tasks: dashboard status counts
CREATE INDEX IF NOT EXISTS idx_receiving_tasks_status ON receiving_tasks(status);
CREATE INDEX IF NOT EXISTS idx_picking_tasks_status ON picking_tasks(status);

-- receiving/picking tasks: created_at for week groupings in dashboard
CREATE INDEX IF NOT EXISTS idx_receiving_tasks_created_at ON receiving_tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_picking_tasks_created_at ON picking_tasks(created_at DESC);
