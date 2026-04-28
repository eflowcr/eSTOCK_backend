package requests

import "encoding/json"

type PatchReceivingTaskRequest struct {
	AssignedTo    *string          `json:"assignedTo,omitempty" validate:"omitempty,max=255"`
	Priority      *string          `json:"priority,omitempty" validate:"omitempty,max=20"`
	Status        *string          `json:"status,omitempty" validate:"omitempty,max=20"`
	Notes         *string          `json:"notes,omitempty" validate:"omitempty,max=1000"`
	Items         *json.RawMessage `json:"items,omitempty"`
	InboundNumber *string          `json:"inboundNumber,omitempty" validate:"omitempty,max=100"`
	// S2 R2 vendor/supplier fields
	SupplierID      *string `json:"supplier_id,omitempty" validate:"omitempty,max=40"`
	VendorRef       *string `json:"vendor_ref,omitempty" validate:"omitempty,max=100"`
	TrackingNumber  *string `json:"tracking_number,omitempty" validate:"omitempty,max=100"`
	ReceptionMethod *string `json:"reception_method,omitempty" validate:"omitempty,max=50"`
	Incoterms       *string `json:"incoterms,omitempty" validate:"omitempty,max=20"`
}

// LinkSupplierRequest is the body for PATCH /receiving-tasks/:id/supplier
type LinkSupplierRequest struct {
	SupplierID *string `json:"supplier_id"` // nil = unlink
}
