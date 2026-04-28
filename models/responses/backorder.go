package responses

import "time"

// BackorderResponse is the full API response for a single backorder.
type BackorderResponse struct {
	ID                     string     `json:"id"`
	TenantID               string     `json:"-"`
	OriginalSalesOrderID   string     `json:"original_sales_order_id"`
	ArticleSKU             string     `json:"article_sku"`
	RemainingQty           float64    `json:"remaining_qty"`
	Status                 string     `json:"status"`
	GeneratedPickingTaskID *string    `json:"generated_picking_task_id,omitempty"`
	FulfilledAt            *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// BackorderListResponse wraps list + pagination.
type BackorderListResponse struct {
	Items      []BackorderResponse `json:"items"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

// FulfillBackorderResult is returned when a backorder fulfillment picking task is created.
type FulfillBackorderResult struct {
	Backorder     *BackorderResponse `json:"backorder"`
	PickingTaskID string             `json:"picking_task_id"`
}
