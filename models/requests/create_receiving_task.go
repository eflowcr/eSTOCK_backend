package requests

import (
	"encoding/json"
)

type CreateReceivingTaskRequest struct {
	InboundNumber string          `json:"inbound_number" binding:"required"`
	AssignedTo    *string         `json:"assigned_to" binding:"omitempty"`
	Priority      string          `json:"priority"`
	Status        *string         `json:"status" binding:"omitempty"`
	Notes         *string         `json:"notes" binding:"omitempty"`
	Items         json.RawMessage `gorm:"column:items;type:jsonb" json:"items"`
}
