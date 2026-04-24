package requests

import "time"

// UpdatePurchaseOrderRequest is the body for PATCH /api/purchase-orders/:id.
// Only editable while status = 'draft'. Other fields are server-stamped.
type UpdatePurchaseOrderRequest struct {
	ExpectedDate *time.Time `json:"expected_date,omitempty"`
	Notes        *string    `json:"notes,omitempty" validate:"omitempty,max=1000"`
}
