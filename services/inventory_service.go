package services

import (
	"sort"
	"strings"

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

func (s *InventoryService) ImportInventoryFromExcel(userId string, fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return s.Repository.ImportInventoryFromExcel(userId, fileBytes)
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

// GetPickSuggestionsBySKU returns pick suggestions for a SKU sorted by: (1) rotation (FIFO/FEFO), (2) quantity ascending.
func (s *InventoryService) GetPickSuggestionsBySKU(sku string) ([]dto.PickSuggestion, *responses.InternalResponse) {
	list, resp := s.Repository.GetPickSuggestionsBySKU(sku)
	if resp != nil || len(list) == 0 {
		return list, resp
	}
	rotationStrategy := "fifo"
	if s.ArticlesRepo != nil {
		article, errResp := s.ArticlesRepo.GetBySku(sku)
		if errResp == nil && article != nil {
			rotationStrategy = strings.TrimSpace(strings.ToLower(article.RotationStrategy))
			if rotationStrategy != "fifo" && rotationStrategy != "fefo" {
				rotationStrategy = "fifo"
			}
		}
	}
	sortPickSuggestions(list, rotationStrategy)
	return list, nil
}

// sortPickSuggestions orders by (1) rotation (FIFO/FEFO) on lot, (2) quantity ascending (lowest first).
func sortPickSuggestions(list []dto.PickSuggestion, strategy string) {
	if strategy == "fefo" {
		sort.Slice(list, func(i, j int) bool {
			ei, ej := list[i].ExpirationDate, list[j].ExpirationDate
			if ei == nil && ej == nil {
				if !list[i].LotCreatedAt.Equal(list[j].LotCreatedAt) {
					return list[i].LotCreatedAt.Before(list[j].LotCreatedAt)
				}
				return list[i].Quantity < list[j].Quantity
			}
			if ei == nil {
				return false
			}
			if ej == nil {
				return true
			}
			if ei.Before(*ej) {
				return true
			}
			if ej.Before(*ei) {
				return false
			}
			if list[i].Quantity != list[j].Quantity {
				return list[i].Quantity < list[j].Quantity
			}
			return list[i].LotCreatedAt.Before(list[j].LotCreatedAt)
		})
		return
	}
	// FIFO: oldest lot first, then lowest quantity first
	sort.Slice(list, func(i, j int) bool {
		if list[i].LotCreatedAt.Before(list[j].LotCreatedAt) {
			return true
		}
		if list[j].LotCreatedAt.Before(list[i].LotCreatedAt) {
			return false
		}
		return list[i].Quantity < list[j].Quantity
	})
}

func (s *InventoryService) GenerateImportTemplate(language string) ([]byte, error) {
	return s.Repository.GenerateImportTemplate(language)
}
