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

func (s *InventoryService) CreateInventory(userId string, item *requests.CreateInventory) *responses.InternalResponse {
	return s.Repository.CreateInventory(userId, item)
}

func (s *InventoryService) UpdateInventory(item *requests.UpdateInventory) *responses.InternalResponse {
	return s.Repository.UpdateInventory(item)
}

func (s *InventoryService) DeleteInventory(sku, location string) *responses.InternalResponse {
	return s.Repository.DeleteInventory(sku, location)
}

func (s *InventoryService) Trend(sku string) (*dto.ConsumptionTrend, *responses.InternalResponse) {
	return s.Repository.Trend(sku)
}

func (s *InventoryService) ImportInventoryFromExcel(userId string, fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return s.Repository.ImportInventoryFromExcel(userId, fileBytes)
}

func (s *InventoryService) ExportInventoryToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportInventoryToExcel()
}

func (s *InventoryService) GetInventoryLots(inventoryID int) ([]responses.InventoryLot, *responses.InternalResponse) {
	return s.Repository.GetInventoryLots(inventoryID)
}

func (s *InventoryService) GetInventorySerials(inventoryID int) ([]responses.InventorySerialWithSerial, *responses.InternalResponse) {
	return s.Repository.GetInventorySerials(inventoryID)
}

func (s *InventoryService) CreateInventoryLot(id int, input *requests.CreateInventoryLotRequest) *responses.InternalResponse {
	return s.Repository.CreateInventoryLot(id, input)
}

func (s *InventoryService) DeleteInventoryLot(id int) *responses.InternalResponse {
	return s.Repository.DeleteInventoryLot(id)
}

func (s *InventoryService) CreateInventorySerial(id int, input *requests.CreateInventorySerial) *responses.InternalResponse {
	return s.Repository.CreateInventorySerial(id, input)
}

func (s *InventoryService) DeleteInventorySerial(id int) *responses.InternalResponse {
	return s.Repository.DeleteInventorySerial(id)
}
