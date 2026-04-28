package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// InventoryCountsRepository defines persistence operations for inventory count sheets.
type InventoryCountsRepository interface {
	List(status, locationID string) ([]database.InventoryCount, *responses.InternalResponse)
	GetByID(id string) (*database.InventoryCount, *responses.InternalResponse)
	GetDetail(id string) (*responses.InventoryCountDetail, *responses.InternalResponse)
	Create(userID string, req *requests.CreateInventoryCount) (*database.InventoryCount, *responses.InternalResponse)
	UpdateStatus(id, status string) *responses.InternalResponse
	MarkStarted(id string) *responses.InternalResponse
	MarkCancelled(id string) *responses.InternalResponse
	MarkSubmitted(id, submittedBy, adjustmentID string) *responses.InternalResponse

	// Lines + locations
	ListLines(countID string) ([]database.InventoryCountLine, *responses.InternalResponse)
	AddLine(line *database.InventoryCountLine) *responses.InternalResponse
	ListLocations(countID string) ([]database.InventoryCountLocation, *responses.InternalResponse)

	// Helpers used during scan-line:
	ResolveSKUByBarcode(barcode string) (string, *responses.InternalResponse)
	GetExpectedQty(sku, locationCode, lot string) (float64, *responses.InternalResponse)
	GetLocationCodeByID(locationID string) (string, *responses.InternalResponse)
}
