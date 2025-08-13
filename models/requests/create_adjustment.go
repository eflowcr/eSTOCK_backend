package requests

import "time"

type CreateAdjustment struct {
	SKU                string          `json:"sku"`
	Location           string          `json:"location"`
	AdjustmentQuantity float64         `json:"adjustment_quantity"`
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
