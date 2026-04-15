package database

// ReceivingTaskItem represents one SKU line in a receiving task.
// Receiving remains single-location per line — unlike picking (B1), the operator
// designates one destination location per item. LotEntry is defined in picking_task_item.go.
type ReceivingTaskItem struct {
	SKU              string     `json:"sku"`
	ExpectedQuantity float64    `json:"expected_qty"`
	ReceivedQuantity *float64   `json:"received_qty,omitempty"`
	Location         string     `json:"location"` // single destination location per line
	Status           *string    `json:"status,omitempty"`
	LotNumbers       []LotEntry `json:"lots,omitempty"`
	SerialNumbers    []Serial   `json:"serials,omitempty"`
}
