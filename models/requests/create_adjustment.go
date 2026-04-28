package requests

import "time"

type CreateAdjustment struct {
	SKU                string          `json:"sku" validate:"required,max=100"`
	Location           string          `json:"location" validate:"required,max=100"`
	AdjustmentQuantity float64         `json:"adjustment_quantity" validate:"required,gte=0"`
	Reason             string          `json:"reason" validate:"required,max=255"`
	Notes              string          `json:"notes" validate:"max=1000"`
	Lots               []AdjustmentLot `json:"lots,omitempty"`
	Serials            []string        `json:"serials,omitempty"`
	// S2 D1: explicit adjustment type. Defaults to "increase" when empty (backward compat).
	// increase: add stock (existing behavior); decrease: subtract, blocked if violates reserved_qty;
	// count_reconcile: absolute physical count — target qty, always notifies admin.
	AdjustmentType string `json:"adjustment_type,omitempty" validate:"omitempty,oneof=increase decrease count_reconcile"`
}

type AdjustmentLot struct {
	LotNumber      string     `json:"lotNumber" validate:"max=100"`
	Quantity       int        `json:"quantity" validate:"gte=0"`
	ExpirationDate *time.Time `json:"expirationDate,omitempty"`
}
