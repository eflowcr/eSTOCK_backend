package requests

import "github.com/eflowcr/eSTOCK_backend/models/database"

type CreateInventory struct {
	SKU         string             `json:"sku" validate:"required"`
	Name        string             `json:"name" validate:"required"`
	Description *string            `json:"description"`
	Location    string             `json:"location" validate:"required"`
	Quantity    float64            `json:"quantity" validate:"required"`
	UnitPrice   *float64           `json:"unitPrice" validate:"required"`
	Lots        []CreateLotRequest `json:"lots,omitempty"`
	Serials     []database.Serial  `json:"serials,omitempty"`
}
