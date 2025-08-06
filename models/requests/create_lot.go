package requests

import "time"

type CreateLotRequest struct {
	LotNumber      string     `json:"lot_number" binding:"required"`
	SKU            string     `json:"sku" binding:"required"`
	Quantity       float64    `json:"quantity" binding:"required"`
	ExpirationDate *time.Time `json:"expiration_date,omitempty"`
}
