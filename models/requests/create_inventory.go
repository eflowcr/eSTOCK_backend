package requests

import "github.com/eflowcr/eSTOCK_backend/models/database"

type CreateInventory struct {
	SKU         string             `json:"sku" validate:"required,max=100"`
	Name        string             `json:"name" validate:"required,max=255"`
	Description *string            `json:"description" validate:"omitempty,max=2000"`
	Location    string             `json:"location" validate:"required,max=100"`
	Quantity    float64            `json:"quantity" validate:"required,gte=0"`
	UnitPrice   *float64           `json:"unitPrice" validate:"required,gte=0"`
	Lots        []CreateLotRequest `json:"lots,omitempty"`
	Serials     []database.Serial  `json:"serials,omitempty"`
}
