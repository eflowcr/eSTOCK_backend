package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// InventoryRepository defines persistence operations for inventory.
type InventoryRepository interface {
	GetPickSuggestionsBySKU(sku string, qty float64) (*dto.PickSuggestionResponse, *responses.InternalResponse)
	GetAllInventory() ([]*dto.EnhancedInventory, *responses.InternalResponse)
	GetInventoryBySkuAndLocation(sku, location string) (*dto.EnhancedInventory, *responses.InternalResponse)
	CreateInventory(userId string, item *requests.CreateInventory) *responses.InternalResponse
	UpdateInventory(item *requests.UpdateInventory) *responses.InternalResponse
	DeleteInventory(sku, location string) *responses.InternalResponse
	Trend(sku string) (*dto.ConsumptionTrend, *responses.InternalResponse)
	ImportInventoryFromExcel(userId string, fileBytes []byte) ([]string, []string, *responses.InternalResponse)
	ImportInventoryFromJSON(userId string, rows []requests.InventoryImportRow) ([]string, []string, *responses.InternalResponse)
	ValidateImportRows(rows []requests.InventoryImportRow) ([]responses.InventoryValidationResult, *responses.InternalResponse)
	ExportInventoryToExcel() ([]byte, *responses.InternalResponse)
	GetInventoryLots(inventoryID string) ([]responses.InventoryLot, *responses.InternalResponse)
	GetInventorySerials(inventoryID string) ([]responses.InventorySerialWithSerial, *responses.InternalResponse)
	CreateInventoryLot(id string, input *requests.CreateInventoryLotRequest) *responses.InternalResponse
	DeleteInventoryLot(id string) *responses.InternalResponse
	CreateInventorySerial(id string, input *requests.CreateInventorySerial) *responses.InternalResponse
	DeleteInventorySerial(id string) *responses.InternalResponse
	GenerateImportTemplate(language string) ([]byte, error)
}
