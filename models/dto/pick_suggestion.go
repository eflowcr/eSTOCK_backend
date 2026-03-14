package dto

import "time"

// PickSuggestion represents a suggested pick location and lot for outbound shipments.
// Sorted by: (1) inventory rotation (FIFO/FEFO), (2) lowest quantity at location first.
type PickSuggestion struct {
	Location       string     `json:"location"`
	LotID          string     `json:"lot_id"`
	LotNumber      string     `json:"lot_number"`
	Quantity       float64    `json:"quantity"`
	ExpirationDate *time.Time `json:"expiration_date,omitempty"`
	LotCreatedAt   time.Time  `json:"lot_created_at"`
}
