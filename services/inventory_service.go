package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type InventoryService struct {
	Repository   ports.InventoryRepository
	ArticlesRepo ports.ArticlesRepository // optional: when set, GetPickSuggestionsBySKU sorts by rotation (FIFO/FEFO) then quantity
}

func NewInventoryService(repo ports.InventoryRepository, articlesRepo ports.ArticlesRepository) *InventoryService {
	return &InventoryService{
		Repository:   repo,
		ArticlesRepo: articlesRepo,
	}
}

func (s *InventoryService) GetAllInventory() ([]*dto.EnhancedInventory, *responses.InternalResponse) {
	return s.Repository.GetAllInventory()
}

func (s *InventoryService) GetInventoryBySkuAndLocation(sku, location string) (*dto.EnhancedInventory, *responses.InternalResponse) {
	return s.Repository.GetInventoryBySkuAndLocation(sku, location)
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

func (s *InventoryService) ImportInventoryFromExcel(userId string, fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	return s.Repository.ImportInventoryFromExcel(userId, fileBytes)
}

func (s *InventoryService) ImportInventoryFromJSON(userId string, rows []requests.InventoryImportRow) ([]string, []string, *responses.InternalResponse) {
	return s.Repository.ImportInventoryFromJSON(userId, rows)
}

func (s *InventoryService) ValidateImportRows(rows []requests.InventoryImportRow) ([]responses.InventoryValidationResult, *responses.InternalResponse) {
	return s.Repository.ValidateImportRows(rows)
}

func (s *InventoryService) ExportInventoryToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportInventoryToExcel()
}

func (s *InventoryService) GetInventoryLots(inventoryID string) ([]responses.InventoryLot, *responses.InternalResponse) {
	return s.Repository.GetInventoryLots(inventoryID)
}

func (s *InventoryService) GetInventorySerials(inventoryID string) ([]responses.InventorySerialWithSerial, *responses.InternalResponse) {
	return s.Repository.GetInventorySerials(inventoryID)
}

func (s *InventoryService) CreateInventoryLot(id string, input *requests.CreateInventoryLotRequest) *responses.InternalResponse {
	return s.Repository.CreateInventoryLot(id, input)
}

func (s *InventoryService) DeleteInventoryLot(id string) *responses.InternalResponse {
	return s.Repository.DeleteInventoryLot(id)
}

func (s *InventoryService) CreateInventorySerial(id string, input *requests.CreateInventorySerial) *responses.InternalResponse {
	return s.Repository.CreateInventorySerial(id, input)
}

func (s *InventoryService) DeleteInventorySerial(id string) *responses.InternalResponse {
	return s.Repository.DeleteInventorySerial(id)
}

// GetPickSuggestionsBySKU returns FEFO-ordered pick allocations for a SKU.
// Sorting is done in SQL (FEFO cross-location). If qty is 0, all available stock is returned.
func (s *InventoryService) GetPickSuggestionsBySKU(sku string, qty float64) (*dto.PickSuggestionResponse, *responses.InternalResponse) {
	return s.Repository.GetPickSuggestionsBySKU(sku, qty)
}

func (s *InventoryService) GenerateImportTemplate(language string) ([]byte, error) {
	return s.Repository.GenerateImportTemplate(language)
}

// GetValuation returns AVCO-based inventory valuation grouped by article, location, or category.
func (s *InventoryService) GetValuation(groupBy string) (*responses.InventoryValuationResponse, *responses.InternalResponse) {
	switch groupBy {
	case "article", "location", "category":
	default:
		groupBy = "article"
	}
	return s.Repository.GetValuation(groupBy)
}
