package requests

import "time"

// CreatePurchaseOrderItemRequest is a line item in a new purchase order.
type CreatePurchaseOrderItemRequest struct {
	ArticleSKU  string   `json:"article_sku" validate:"required,max=100"`
	ExpectedQty float64  `json:"expected_qty" validate:"required,gt=0"`
	UnitCost    *float64 `json:"unit_cost,omitempty" validate:"omitempty,gte=0"`
	Notes       *string  `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// CreatePurchaseOrderRequest is the body for POST /api/purchase-orders/.
// tenant_id and created_by are stamped server-side — never trusted from body.
type CreatePurchaseOrderRequest struct {
	SupplierID   string                           `json:"supplier_id" validate:"required,max=40"`
	ExpectedDate *time.Time                       `json:"expected_date,omitempty"`
	Notes        *string                          `json:"notes,omitempty" validate:"omitempty,max=1000"`
	Items        []CreatePurchaseOrderItemRequest `json:"items" validate:"required,min=1,dive"`
}
