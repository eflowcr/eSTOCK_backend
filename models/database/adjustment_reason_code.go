package database

import "time"

// AdjustmentReasonCode represents a reason code for stock adjustments (inbound = add, outbound = subtract).
type AdjustmentReasonCode struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Direction    string    `json:"direction"` // "inbound" or "outbound"
	IsSystem     bool      `json:"is_system"`
	DisplayOrder int32     `json:"display_order"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
