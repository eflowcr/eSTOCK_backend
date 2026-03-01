package requests

import "time"

type CreateAdjustment struct {
	SKU                string          `json:"sku" validate:"required"`
	Location           string          `json:"location" validate:"required"`
	AdjustmentQuantity float64         `json:"adjustment_quantity" validate:"required"`
	Reason             string          `json:"reason"`
	Notes              string          `json:"notes"`
	Lots               []AdjustmentLot `json:"lots,omitempty"`
	Serials            []string        `json:"serials,omitempty"`
}

type AdjustmentLot struct {
	LotNumber      string     `json:"lotNumber"`
	Quantity       int        `json:"quantity"`
	ExpirationDate *time.Time `json:"expirationDate,omitempty"`
}
