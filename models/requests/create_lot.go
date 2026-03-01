package requests

type CreateLotRequest struct {
	LotNumber        string   `json:"lot_number" binding:"required" validate:"required"`
	SKU              string   `json:"sku" binding:"required" validate:"required"`
	Quantity         float64  `json:"quantity" binding:"required" validate:"required,gte=0"`
	ReceivedQuantity *float64 `json:"received_quantity,omitempty"`
	ExpirationDate   *string  `json:"expiration_date,omitempty"`
	Status           *string  `json:"status,omitempty"`
}
 