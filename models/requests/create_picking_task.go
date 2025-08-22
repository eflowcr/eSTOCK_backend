package requests

import "encoding/json"

type CreatePickingTaskRequest struct {
	OutboundNumber string          `json:"outbound_number" binding:"required"`
	AssignedTo     *string         `json:"assigned_to" binding:"omitempty"`
	Priority       string          `json:"priority"`
	Notes          *string         `json:"notes" binding:"omitempty"`
	Items          json.RawMessage `gorm:"column:items;type:jsonb" json:"items"`
}
