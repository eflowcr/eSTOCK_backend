package requests

import "encoding/json"

type PatchReceivingTaskRequest struct {
	AssignedTo    *string          `json:"assignedTo,omitempty" validate:"omitempty,max=255"`
	Priority      *string          `json:"priority,omitempty" validate:"omitempty,max=20"`
	Status        *string          `json:"status,omitempty" validate:"omitempty,max=20"`
	Notes         *string          `json:"notes,omitempty" validate:"omitempty,max=1000"`
	Items         *json.RawMessage `json:"items,omitempty"`
	InboundNumber *string          `json:"inboundNumber,omitempty" validate:"omitempty,max=100"`
}
