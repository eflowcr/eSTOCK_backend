package ports

import "github.com/eflowcr/eSTOCK_backend/models/responses"

// BackordersRepository defines persistence operations for backorders (BO1 + BO2).
type BackordersRepository interface {
	// List returns paginated backorders filtered by tenant and optional status.
	List(tenantID string, status, soID *string, page, limit int) (*responses.BackorderListResponse, *responses.InternalResponse)

	// GetByID returns a single backorder scoped to tenantID.
	GetByID(id, tenantID string) (*responses.BackorderResponse, *responses.InternalResponse)

	// Fulfill creates a new picking task linked to the original SO, updates the backorder,
	// and returns the new picking task ID.
	// Enforce max depth=1: only call when backorder.status == 'pending'.
	Fulfill(id, tenantID, userID string) (*responses.FulfillBackorderResult, *responses.InternalResponse)
}
