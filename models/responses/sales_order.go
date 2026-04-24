package responses

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
)

// SalesOrderResponse is the detailed view of a sales order returned by the API.
type SalesOrderResponse struct {
	ID            string                        `json:"id"`
	TenantID      string                        `json:"-"`
	SONumber      string                        `json:"so_number"`
	CustomerID    string                        `json:"customer_id"`
	CustomerName  *string                       `json:"customer_name,omitempty"`
	Status        string                        `json:"status"`
	ExpectedDate  *time.Time                    `json:"expected_date,omitempty"`
	Notes         *string                       `json:"notes,omitempty"`
	CreatedBy     *string                       `json:"created_by,omitempty"`
	SubmittedAt   *time.Time                    `json:"submitted_at,omitempty"`
	CompletedAt   *time.Time                    `json:"completed_at,omitempty"`
	CancelledAt   *time.Time                    `json:"cancelled_at,omitempty"`
	PickingTaskID *string                       `json:"picking_task_id,omitempty"`
	CreatedAt     time.Time                     `json:"created_at"`
	UpdatedAt     time.Time                     `json:"updated_at"`
	Items         []database.SalesOrderItem     `json:"items"`
}

// SalesOrderListItem is the lightweight view used in list responses.
type SalesOrderListItem struct {
	ID            string     `json:"id"`
	SONumber      string     `json:"so_number"`
	CustomerID    string     `json:"customer_id"`
	CustomerName  *string    `json:"customer_name,omitempty"`
	Status        string     `json:"status"`
	ExpectedDate  *time.Time `json:"expected_date,omitempty"`
	SubmittedAt   *time.Time `json:"submitted_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	PickingTaskID *string    `json:"picking_task_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ItemCount     int        `json:"item_count"`
}

// SalesOrderListResponse wraps list + pagination.
type SalesOrderListResponse struct {
	Items      []SalesOrderListItem `json:"items"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
}

// SubmitSalesOrderResult is returned from the submit endpoint.
type SubmitSalesOrderResult struct {
	SalesOrder    *SalesOrderResponse `json:"sales_order"`
	PickingTaskID string              `json:"picking_task_id"`
	// BackorderCandidates lists SKUs with insufficient stock (not blocked — picking created for available items).
	BackorderCandidates []BackorderCandidate `json:"backorder_candidates,omitempty"`
}

// BackorderCandidate describes a line item that couldn't be fully allocated at submit time.
type BackorderCandidate struct {
	ArticleSKU      string  `json:"article_sku"`
	RequestedQty    float64 `json:"requested_qty"`
	AvailableQty    float64 `json:"available_qty"`
	BackorderQty    float64 `json:"backorder_qty"`
}
