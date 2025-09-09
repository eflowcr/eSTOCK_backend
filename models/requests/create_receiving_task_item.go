package requests

import "github.com/eflowcr/eSTOCK_backend/models/database"

type ReceivingTaskItemRequest struct {
	SKU              string             `json:"sku"`
	ExpectedQuantity int                `json:"expected_qty"`
	Location         string             `json:"location"`
	LotNumbers       []CreateLotRequest `json:"lots,omitempty"`
	SerialNumbers    []database.Serial  `json:"serials,omitempty"`
}
