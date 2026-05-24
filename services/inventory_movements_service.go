package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type InventoryMovementsService struct {
	Repository ports.InventoryMovementsRepository
}

func NewInventoryMovementsService(repo ports.InventoryMovementsRepository) *InventoryMovementsService {
	return &InventoryMovementsService{
		Repository: repo,
	}
}

func (s *InventoryMovementsService) GetAllInventoryMovements(sku string) ([]database.InventoryMovement, *responses.InternalResponse) {
	return s.Repository.GetAllInventoryMovements(sku)
}

func (s *InventoryMovementsService) ListMovements(f ports.MovementsFilter) ([]database.InventoryMovement, *responses.InternalResponse) {
	return s.Repository.ListMovements(f)
}

// ListRecentMovements returns the most recent tenant-wide movements (no SKU
// filter), newest first, capped at limit. Backs the mobile Historial feed.
func (s *InventoryMovementsService) ListRecentMovements(tenantID string, limit int) ([]database.InventoryMovement, *responses.InternalResponse) {
	return s.Repository.ListRecentMovements(tenantID, limit)
}
