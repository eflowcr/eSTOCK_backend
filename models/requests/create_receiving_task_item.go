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
	// S2 R1: explicit accepted/rejected split. When both are nil/0 but ReceivedQuantity > 0
	// the service backfills AcceptedQty = ReceivedQuantity (legacy compatibility).
	AcceptedQty *float64 `json:"accepted_qty,omitempty" validate:"omitempty,gte=0"`
	RejectedQty *float64 `json:"rejected_qty,omitempty" validate:"omitempty,gte=0"`
}
