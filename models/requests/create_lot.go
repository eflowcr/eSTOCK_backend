package requests

type CreateLotRequest struct {
	LotNumber        string   `json:"lot_number" binding:"required"`
	SKU              string   `json:"sku" binding:"required"`
	Quantity         float64  `json:"quantity" binding:"required"`
	ReceivedQuantity *float64 `json:"received_quantity" binding:"required"`
	ExpirationDate   *string  `json:"expiration_date,omitempty"`
	Status           *string  `json:"status,omitempty"`
}
