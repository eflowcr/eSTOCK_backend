package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// InventoryMovementsRepository defines persistence operations for inventory movements.
type InventoryMovementsRepository interface {
	GetAllInventoryMovements(sku string) ([]database.InventoryMovement, *responses.InternalResponse)
}
