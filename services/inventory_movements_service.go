package services

import (
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

func (s *InventoryMovementsService) GetAllInventoryMovements() ([]responses.InventoryMovementView, *responses.InternalResponse) {
	return s.Repository.GetAllInventoryMovements()
}

func (s *InventoryMovementsService) GetMovementsBySku(sku string) ([]responses.InventoryMovementView, *responses.InternalResponse) {
	return s.Repository.GetMovementsBySku(sku)
}
