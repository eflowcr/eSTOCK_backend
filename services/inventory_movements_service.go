package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type InventoryMovementsService struct {
	Repository *repositories.InventoryMovementsRepository
}

func NewInventoryMovementsService(repo *repositories.InventoryMovementsRepository) *InventoryMovementsService {
	return &InventoryMovementsService{
		Repository: repo,
	}
}

func (s *InventoryMovementsService) GetAllInventoryMovements(sku string) ([]database.InventoryMovement, *responses.InternalResponse) {
	return s.Repository.GetAllInventoryMovements(sku)
}
