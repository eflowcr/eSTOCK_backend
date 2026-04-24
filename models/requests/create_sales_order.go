package requests

import "time"

// CreateSalesOrderRequest is the body for POST /api/sales-orders.
type CreateSalesOrderRequest struct {
	CustomerID   string                   `json:"customer_id" validate:"required,max=40"`
	ExpectedDate *time.Time               `json:"expected_date,omitempty"`
	Notes        *string                  `json:"notes,omitempty" validate:"omitempty,max=2000"`
	Items        []CreateSalesOrderItem   `json:"items" validate:"required,min=1,dive"`
}

// CreateSalesOrderItem is a line item inside CreateSalesOrderRequest.
type CreateSalesOrderItem struct {
	ArticleSKU  string   `json:"article_sku" validate:"required,max=100"`
	ExpectedQty float64  `json:"expected_qty" validate:"required,gt=0"`
	UnitPrice   *float64 `json:"unit_price,omitempty" validate:"omitempty,gte=0"`
	Notes       *string  `json:"notes,omitempty" validate:"omitempty,max=1000"`
}

// UpdateSalesOrderRequest is the body for PATCH /api/sales-orders/:id (draft only).
type UpdateSalesOrderRequest struct {
	CustomerID   *string                  `json:"customer_id,omitempty" validate:"omitempty,max=40"`
	ExpectedDate *time.Time               `json:"expected_date,omitempty"`
	Notes        *string                  `json:"notes,omitempty" validate:"omitempty,max=2000"`
	Items        []CreateSalesOrderItem   `json:"items,omitempty" validate:"omitempty,dive"`
}
