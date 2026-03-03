package requests

import (
	"encoding/json"
)

type CreateReceivingTaskRequest struct {
	InboundNumber string          `json:"inbound_number" binding:"required" validate:"required,max=100"`
	AssignedTo    *string         `json:"assigned_to,omitempty" validate:"omitempty,max=255"`
	Priority      string          `json:"priority" validate:"max=20"`
	Status        *string         `json:"status,omitempty" validate:"omitempty,max=20"`
	Notes         *string         `json:"notes,omitempty" validate:"omitempty,max=1000"`
	Items         json.RawMessage `gorm:"column:items;type:jsonb" json:"items" validate:"required"`
}
