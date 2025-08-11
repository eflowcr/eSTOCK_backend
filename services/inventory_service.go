package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type InventoryService struct {
	Repository *repositories.InventoryRepository
}

func NewInventoryService(repo *repositories.InventoryRepository) *InventoryService {
	return &InventoryService{
		Repository: repo,
	}
}

func (s *InventoryService) GetAllInventory() ([]*dto.EnhancedInventory, *responses.InternalResponse) {
	return s.Repository.GetAllInventory()
}

func (s *InventoryService) CreateInventory(item *requests.CreateInventory) *responses.InternalResponse {
	return s.Repository.CreateInventory(item)
}
