package requests

import "encoding/json"

type CreatePickingTaskRequest struct {
	OutboundNumber string          `json:"outbound_number" binding:"required" validate:"required,max=100"`
	AssignedTo     *string         `json:"assigned_to,omitempty" validate:"omitempty,max=255"`
	Priority       string          `json:"priority" validate:"max=20"`
	Notes          *string         `json:"notes,omitempty" validate:"omitempty,max=1000"`
	Items          json.RawMessage `gorm:"column:items;type:jsonb" json:"items" validate:"required"`
}
