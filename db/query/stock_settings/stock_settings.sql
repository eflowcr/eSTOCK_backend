-- StockSettings CRUD for sqlc
-- Schema: db/migrations/000018_sprint_s2.up.sql (stock_settings table)

-- name: GetStockSettings :one
SELECT tenant_id, valuation_method, pick_batch_based_on, over_receipt_allowance_pct,
  over_delivery_allowance_pct, over_picking_allowance_pct, auto_reserve_stock,
  allow_partial_reservation, expiry_alert_days, auto_create_material_request,
  partial_delivery_policy, updated_at
FROM stock_settings WHERE tenant_id = $1;

-- name: UpsertStockSettings :one
INSERT INTO stock_settings (
  tenant_id, valuation_method, pick_batch_based_on, over_receipt_allowance_pct,
  over_delivery_allowance_pct, over_picking_allowance_pct, auto_reserve_stock,
  allow_partial_reservation, expiry_alert_days, auto_create_material_request,
  partial_delivery_policy
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (tenant_id) DO UPDATE SET
  valuation_method = EXCLUDED.valuation_method,
  pick_batch_based_on = EXCLUDED.pick_batch_based_on,
  over_receipt_allowance_pct = EXCLUDED.over_receipt_allowance_pct,
  over_delivery_allowance_pct = EXCLUDED.over_delivery_allowance_pct,
  over_picking_allowance_pct = EXCLUDED.over_picking_allowance_pct,
  auto_reserve_stock = EXCLUDED.auto_reserve_stock,
  allow_partial_reservation = EXCLUDED.allow_partial_reservation,
  expiry_alert_days = EXCLUDED.expiry_alert_days,
  auto_create_material_request = EXCLUDED.auto_create_material_request,
  partial_delivery_policy = EXCLUDED.partial_delivery_policy,
  updated_at = now()
RETURNING tenant_id, valuation_method, pick_batch_based_on, over_receipt_allowance_pct,
  over_delivery_allowance_pct, over_picking_allowance_pct, auto_reserve_stock,
  allow_partial_reservation, expiry_alert_days, auto_create_material_request,
  partial_delivery_policy, updated_at;
