package requests

import "github.com/eflowcr/eSTOCK_backend/models/database"

type ReceivingTaskItemRequest struct {
	SKU              string             `json:"sku" validate:"required"`
	ExpectedQuantity int                `json:"expected_qty" validate:"gte=0"`
	Location         string             `json:"location" validate:"required"`
	LotNumbers       []CreateLotRequest `json:"lots,omitempty"`
	SerialNumbers    []database.Serial  `json:"serials,omitempty"`
	Status           *string            `json:"status,omitempty"`
	ReceivedQuantity *int               `json:"received_qty,omitempty"`
}
