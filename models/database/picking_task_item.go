package database

// LotEntry represents a structured lot with quantity and expiration.
// Shared between picking and receiving task items.
type LotEntry struct {
	LotNumber      string  `json:"lot_number"`
	SKU            string  `json:"sku,omitempty"`
	Quantity       float64 `json:"quantity"`
	ExpirationDate *string `json:"expiration_date,omitempty"`
	Status         *string `json:"status,omitempty"` // pending | picked | received | skipped
}

// LocationAllocation indicates from which location and in what quantity to pick.
// ExpirationDate is display-only — populated from pick-suggestions, not persisted in picking_tasks.
type LocationAllocation struct {
	Location       string   `json:"location"`
	Quantity       float64  `json:"quantity"`
	LotNumber      *string  `json:"lot_number,omitempty"`
	PickedQty      *float64 `json:"picked_qty,omitempty"`
	Status         *string  `json:"status,omitempty"` // pending | picked | skipped
	ExpirationDate *string  `json:"expiration_date,omitempty"` // "YYYY-MM-DD" — display only
}

type PickingTaskItem struct {
	SKU              string               `json:"sku"`
	ExpectedQuantity float64              `json:"required_qty"`
	Allocations      []LocationAllocation `json:"allocations"` // replaces Location string (A1)
	Status           *string              `json:"status,omitempty"`
	PickedQty        *float64             `json:"picked_qty,omitempty"`
	LotNumbers       []LotEntry           `json:"lots,omitempty"`
	SerialNumbers    []Serial             `json:"serials,omitempty"`
}
