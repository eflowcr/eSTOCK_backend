package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// InventoryRepository defines persistence operations for inventory.
type InventoryRepository interface {
	GetAllInventory() ([]*dto.EnhancedInventory, *responses.InternalResponse)
	CreateInventory(userId string, item *requests.CreateInventory) *responses.InternalResponse
	UpdateInventory(item *requests.UpdateInventory) *responses.InternalResponse
	DeleteInventory(sku, location string) *responses.InternalResponse
	Trend(sku string) (*dto.ConsumptionTrend, *responses.InternalResponse)
	ImportInventoryFromExcel(userId string, fileBytes []byte) ([]string, []*responses.InternalResponse)
	ExportInventoryToExcel() ([]byte, *responses.InternalResponse)
	GetInventoryLots(inventoryID int) ([]responses.InventoryLot, *responses.InternalResponse)
	GetInventorySerials(inventoryID int) ([]responses.InventorySerialWithSerial, *responses.InternalResponse)
	CreateInventoryLot(id int, input *requests.CreateInventoryLotRequest) *responses.InternalResponse
	DeleteInventoryLot(id int) *responses.InternalResponse
	CreateInventorySerial(id int, input *requests.CreateInventorySerial) *responses.InternalResponse
	DeleteInventorySerial(id int) *responses.InternalResponse
}
