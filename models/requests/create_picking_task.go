package requests

import "encoding/json"

type CreatePickingTaskRequest struct {
	OutboundNumber string          `json:"outboundNumber" binding:"required"`
	AssignedTo     *string         `json:"assignedTo" binding:"omitempty"`
	Priority       string          `json:"priority"`
	Notes          *string         `json:"notes" binding:"omitempty"`
	Items          json.RawMessage `gorm:"column:items;type:jsonb" json:"items"`
}
