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
}
