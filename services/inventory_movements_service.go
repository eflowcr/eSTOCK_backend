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
