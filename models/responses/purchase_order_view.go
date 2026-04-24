package responses

import "time"

// PurchaseOrderItemView is the embedded line item for purchase order responses.
type PurchaseOrderItemView struct {
	ID              string   `json:"id"`
	ArticleSKU      string   `json:"article_sku"`
	ExpectedQty     float64  `json:"expected_qty"`
	ReceivedQty     float64  `json:"received_qty"`
	RejectedQty     float64  `json:"rejected_qty"`
	Discrepancy     *float64 `json:"discrepancy,omitempty"`
	UnitCost        *float64 `json:"unit_cost,omitempty"`
	Notes           *string  `json:"notes,omitempty"`
}

// PurchaseOrderView is the response shape for purchase order endpoints.
// TenantID is hidden from JSON responses (json:"-").
type PurchaseOrderView struct {
	ID              string                  `json:"id"`
	PONumber        string                  `json:"po_number"`
	SupplierID      string                  `json:"supplier_id"`
	Status          string                  `json:"status"`
	ExpectedDate    *time.Time              `json:"expected_date,omitempty"`
	Notes           *string                 `json:"notes,omitempty"`
	CreatedBy       *string                 `json:"created_by,omitempty"`
	SubmittedAt     *time.Time              `json:"submitted_at,omitempty"`
	CompletedAt     *time.Time              `json:"completed_at,omitempty"`
	CancelledAt     *time.Time              `json:"cancelled_at,omitempty"`
	ReceivingTaskID *string                 `json:"receiving_task_id,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
	Items           []PurchaseOrderItemView `json:"items,omitempty"`
	// Tenant isolation — never leak UUID in HTTP responses.
	TenantID string `json:"-" gorm:"column:tenant_id"`
}

// PurchaseOrderSubmitResponse adds the new receiving_task_id to the submit response.
type PurchaseOrderSubmitResponse struct {
	PurchaseOrderView
	NewReceivingTaskID string `json:"new_receiving_task_id"`
}
