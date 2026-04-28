package requests

type UpdateStockSettingsRequest struct {
	ValuationMethod           string  `json:"valuation_method" binding:"required" validate:"required,oneof=avco fifo"`
	PickBatchBasedOn          string  `json:"pick_batch_based_on" binding:"required" validate:"required,oneof=fefo fifo lifo"`
	OverReceiptAllowancePct   float64 `json:"over_receipt_allowance_pct" validate:"gte=0,lte=100"`
	OverDeliveryAllowancePct  float64 `json:"over_delivery_allowance_pct" validate:"gte=0,lte=100"`
	OverPickingAllowancePct   float64 `json:"over_picking_allowance_pct" validate:"gte=0,lte=100"`
	AutoReserveStock          bool    `json:"auto_reserve_stock"`
	AllowPartialReservation   bool    `json:"allow_partial_reservation"`
	ExpiryAlertDays           int     `json:"expiry_alert_days" validate:"gte=0"`
	AutoCreateMaterialRequest bool    `json:"auto_create_material_request"`
	PartialDeliveryPolicy     string  `json:"partial_delivery_policy" binding:"required" validate:"required,oneof=immediate when_all_ready"`
}
