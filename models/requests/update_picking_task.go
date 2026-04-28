package requests

import "encoding/json"

type PatchPickingTaskRequest struct {
	AssignedTo     *string          `json:"assignedTo,omitempty" validate:"omitempty,max=255"`
	Priority       *string          `json:"priority,omitempty" validate:"omitempty,max=20"`
	Status         *string          `json:"status,omitempty" validate:"omitempty,max=20"`
	Notes          *string          `json:"notes,omitempty" validate:"omitempty,max=1000"`
	Items          *json.RawMessage `json:"items,omitempty"`
	OutboundNumber *string          `json:"outboundNumber,omitempty" validate:"omitempty,max=100"`
	// S2 R2 customer field
	CustomerID *string `json:"customer_id,omitempty" validate:"omitempty,max=40"`
}

// LinkCustomerRequest is the body for PATCH /picking-tasks/:id/customer
type LinkCustomerRequest struct {
	CustomerID *string `json:"customer_id"` // nil = unlink
}
