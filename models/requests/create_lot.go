package requests

type CreateLotRequest struct {
	LotNumber        string   `json:"lot_number" binding:"required" validate:"required,max=100"`
	SKU              string   `json:"sku" binding:"required" validate:"required,max=100"`
	Quantity         float64  `json:"quantity" binding:"required" validate:"required,gte=0"`
	ReceivedQuantity *float64 `json:"received_quantity,omitempty" validate:"omitempty,gte=0"`
	ExpirationDate   *string  `json:"expiration_date,omitempty"`
	Status           *string  `json:"status,omitempty" validate:"omitempty,max=20"`
	// M2 extended fields
	LotNotes       *string `json:"lot_notes,omitempty"`
	ManufacturedAt *string `json:"manufactured_at,omitempty"`
	BestBeforeDate *string `json:"best_before_date,omitempty"`
}
 