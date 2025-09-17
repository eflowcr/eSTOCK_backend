package requests

import "github.com/eflowcr/eSTOCK_backend/models/database"

type PickingTaskItemRequest struct {
	SKU               string             `json:"sku"`
	ExpectedQuantity  int                `json:"required_qty"`
	Location          string             `json:"location"`
	LotNumbers        []CreateLotRequest `json:"lots,omitempty"`
	SerialNumbers     []database.Serial  `json:"serials,omitempty"`
	Status            *string            `json:"status,omitempty"`
	DeliveredQuantity *int               `json:"delivered_qty,omitempty"`
}
