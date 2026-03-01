package requests

import (
	"encoding/json"
)

type CreateReceivingTaskRequest struct {
	InboundNumber string          `json:"inbound_number" binding:"required" validate:"required,max=100"`
	AssignedTo    *string         `json:"assigned_to,omitempty"`
	Priority      string          `json:"priority"`
	Status        *string         `json:"status,omitempty"`
	Notes         *string         `json:"notes,omitempty"`
	Items         json.RawMessage `gorm:"column:items;type:jsonb" json:"items" validate:"required"`
}
