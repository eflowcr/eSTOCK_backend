package dto

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
)

// PickSuggestion is the legacy shape returned before H3.
// Retained for reference; no longer used by active endpoints.
type PickSuggestion struct {
	Location       string     `json:"location"`
	LotID          string     `json:"lot_id"`
	LotNumber      string     `json:"lot_number"`
	Quantity       float64    `json:"quantity"`
	ExpirationDate *time.Time `json:"expiration_date,omitempty"`
	LotCreatedAt   time.Time  `json:"lot_created_at"`
}

// PickSuggestionResponse is the H3 contract for GET /inventory/pick-suggestions/:sku?qty=N.
// Each Allocation is a (location, lot, qty) tuple the operator should pull from.
// Items are FEFO-sorted (earliest expiration first, FIFO within tie).
// If qty is 0, all available allocations are returned (Sufficient is always true).
type PickSuggestionResponse struct {
	Allocations []database.LocationAllocation `json:"allocations"`
	TotalFound  float64                        `json:"total_found"`
	Requested   float64                        `json:"requested_qty"`
	Sufficient  bool                           `json:"sufficient"`
}
