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
	// S2 R2 vendor/supplier fields
	SupplierID      *string `json:"supplier_id,omitempty" validate:"omitempty,max=40"`
	VendorRef       *string `json:"vendor_ref,omitempty" validate:"omitempty,max=100"`
	TrackingNumber  *string `json:"tracking_number,omitempty" validate:"omitempty,max=100"`
	ReceptionMethod *string `json:"reception_method,omitempty" validate:"omitempty,max=50"`
	Incoterms       *string `json:"incoterms,omitempty" validate:"omitempty,max=20"`
}
