package requests

import "encoding/json"

type CreatePickingTaskRequest struct {
	OutboundNumber string          `json:"outbound_number" binding:"required" validate:"required,max=100"`
	AssignedTo     *string         `json:"assigned_to,omitempty"`
	Priority       string          `json:"priority"`
	Notes          *string         `json:"notes,omitempty"`
	Items          json.RawMessage `gorm:"column:items;type:jsonb" json:"items" validate:"required"`
}
