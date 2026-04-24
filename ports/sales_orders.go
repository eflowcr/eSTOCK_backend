package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// SalesOrdersRepository defines persistence operations for sales orders.
type SalesOrdersRepository interface {
	// SO1 — CRUD

	// Create inserts a new draft sales order with its line items.
	// Server-stamps tenant_id, created_by, and so_number.
	Create(tenantID, userID string, req *requests.CreateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse)

	// List returns paginated sales orders with optional filters.
	List(tenantID string, status, customerID, search *string, dateFrom, dateTo *string, page, limit int) (*responses.SalesOrderListResponse, *responses.InternalResponse)

	// GetByID returns the full sales order (header + items) scoped to tenantID.
	GetByID(id, tenantID string) (*responses.SalesOrderResponse, *responses.InternalResponse)

	// Update patches a draft sales order (fields + items).
	// Returns error if status != 'draft'.
	Update(id, tenantID string, req *requests.UpdateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse)

	// SoftDelete marks deleted_at (only when status == 'draft').
	SoftDelete(id, tenantID string) *responses.InternalResponse

	// SO2 — Lifecycle

	// Submit transitions draft → submitted, auto-generates picking task with FEFO suggestions.
	Submit(id, tenantID, userID string) (*responses.SubmitSalesOrderResult, *responses.InternalResponse)

	// Cancel transitions any non-completed status → cancelled; releases picking reservations.
	Cancel(id, tenantID, userID string) *responses.InternalResponse

	// SO3 — Picking auto-link (called from picking_task_repository after complete)

	// UpdatePickedQty updates sales_order_items.picked_qty after picking completion and
	// advances SO status to 'completed' or 'partial' accordingly.
	// Returns the new SO status string for DN/BO routing in CompletePickingTask.
	UpdatePickedQty(salesOrderID string, pickedPerSKU map[string]float64) (string, *responses.InternalResponse)
}
