package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// PurchaseOrdersRepository defines persistence operations for purchase orders.
// All operations are tenant-scoped.
type PurchaseOrdersRepository interface {
	// Create inserts a new draft PO and its items inside a transaction.
	// po_number is auto-generated (PO-YYYY-NNNN) with a row lock.
	// created_by and tenant_id are injected by the caller (from JWT / Config).
	Create(tenantID, createdBy string, req *requests.CreatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse)

	// GetByID returns a PO with embedded items, scoped to tenantID.
	GetByID(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse)

	// List returns POs for a tenant with optional filters and pagination.
	List(tenantID string, status, supplierID, search *string, from, to *string, limit, offset int) ([]responses.PurchaseOrderView, *responses.InternalResponse)

	// Update patches mutable fields on a draft PO (status must be 'draft').
	Update(id, tenantID string, req *requests.UpdatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse)

	// SoftDelete sets deleted_at on a PO, scoped to tenantID.
	SoftDelete(id, tenantID string) *responses.InternalResponse

	// Submit transitions a draft PO to 'submitted' and auto-generates a receiving task.
	// Returns the updated PO view and the newly created receiving task ID.
	Submit(id, tenantID, userID string) (*responses.PurchaseOrderView, string, *responses.InternalResponse)

	// Cancel transitions a non-completed PO to 'cancelled'.
	Cancel(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse)

	// UpdateReceivedQty is called by the receiving completion flow to update
	// purchase_order_items.received_qty / rejected_qty after a receiving task line is completed.
	// If all items are fulfilled, the PO status advances to 'completed'; if partial, to 'partial'.
	UpdateReceivedQty(purchaseOrderID string, itemUpdates []database.PurchaseOrderItemQtyUpdate) *responses.InternalResponse
}
