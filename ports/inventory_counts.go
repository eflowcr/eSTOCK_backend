package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
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

	// SubmitWithAdjustments fans out one adjustment per non-zero variance line and
	// flips the count to "submitted" inside a single GORM transaction (atomicity —
	// W0 hostile-review N1-2). The creator argument is responsible for applying the
	// reason-code-driven sign flip; the repo only orchestrates the transaction.
	SubmitWithAdjustments(countID, userID string, creator InventoryAdjustmentsCreator) *responses.InternalResponse

	// Lines + locations
	ListLines(countID string) ([]database.InventoryCountLine, *responses.InternalResponse)
	AddLine(line *database.InventoryCountLine) *responses.InternalResponse
	ListLocations(countID string) ([]database.InventoryCountLocation, *responses.InternalResponse)

	// Helpers used during scan-line:
	ResolveSKUByBarcode(barcode string) (string, *responses.InternalResponse)
	GetExpectedQty(sku, locationCode, lot string) (float64, *responses.InternalResponse)
	GetLocationCodeByID(locationID string) (string, *responses.InternalResponse)
}

// InventoryAdjustmentsCreator is the narrow surface that the inventory-counts
// submit pipeline needs from the adjustments service. Implementations are
// responsible for reason-code lookup + sign flipping (see AdjustmentsService).
//
// Defined as a port so the inventory-counts service can depend on the abstraction
// rather than the concrete *services.AdjustmentsService — keeps tests cheap.
type InventoryAdjustmentsCreator interface {
	CreateAdjustmentTx(tx *gorm.DB, userId string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse)
}
