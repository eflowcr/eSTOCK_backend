package requests

import "encoding/json"

type PatchReceivingTaskRequest struct {
	AssignedTo    *string          `json:"assignedTo,omitempty"`
	Priority      *string          `json:"priority,omitempty"`
	Status        *string          `json:"status,omitempty"`
	Notes         *string          `json:"notes,omitempty"`
	Items         *json.RawMessage `json:"items,omitempty"`
	InboundNumber *string          `json:"inboundNumber,omitempty"`
}
