package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// MovementsFilter holds optional query filters for listing inventory movements.
type MovementsFilter struct {
	SKU           string
	Location      string
	LotID         string
	MovementType  string
	ReferenceType string
	UserID        string
	From          string
	To            string
	Limit         int
	Offset        int
}

// InventoryMovementsRepository defines persistence operations for inventory movements.
type InventoryMovementsRepository interface {
	GetAllInventoryMovements(sku string) ([]database.InventoryMovement, *responses.InternalResponse)
	ListMovements(f MovementsFilter) ([]database.InventoryMovement, *responses.InternalResponse)
	// ListRecentMovements returns the most recent movements tenant-wide (no SKU
	// filter), newest first, capped at limit. inventory_movements has no
	// tenant_id column of its own (only operational tables got one in
	// migration 000019), so tenant scoping is achieved by joining against
	// articles (tenant-isolated since migration 000029) on sku. This is the
	// recent-all feed the mobile Historial tab consumes.
	ListRecentMovements(tenantID string, limit int) ([]database.InventoryMovement, *responses.InternalResponse)
}
