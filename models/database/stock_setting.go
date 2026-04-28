package database

import "time"

type StockSetting struct {
	TenantID                  string    `json:"tenant_id"`
	ValuationMethod           string    `json:"valuation_method"`
	PickBatchBasedOn          string    `json:"pick_batch_based_on"`
	OverReceiptAllowancePct   float64   `json:"over_receipt_allowance_pct"`
	OverDeliveryAllowancePct  float64   `json:"over_delivery_allowance_pct"`
	OverPickingAllowancePct   float64   `json:"over_picking_allowance_pct"`
	AutoReserveStock          bool      `json:"auto_reserve_stock"`
	AllowPartialReservation   bool      `json:"allow_partial_reservation"`
	ExpiryAlertDays           int       `json:"expiry_alert_days"`
	AutoCreateMaterialRequest bool      `json:"auto_create_material_request"`
	PartialDeliveryPolicy     string    `json:"partial_delivery_policy"`
	UpdatedAt                 time.Time `json:"updated_at"`
}
