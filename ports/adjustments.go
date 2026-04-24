package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// AdjustmentsRepository defines persistence operations for adjustments.
type AdjustmentsRepository interface {
	// GetAllAdjustments returns all adjustments without tenant filter.
	// internal use only — bypass tenant. Prefer GetAllForTenant in HTTP handlers.
	GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse)
	// GetAllForTenant returns adjustments scoped to a specific tenant (S2.5 M3.1).
	GetAllForTenant(tenantID string) ([]database.Adjustment, *responses.InternalResponse)
	GetAdjustmentByID(id string) (*database.Adjustment, *responses.InternalResponse)
	GetAdjustmentDetails(id string) (*dto.AdjustmentDetails, *responses.InternalResponse)
	CreateAdjustment(userId string, tenantID string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse)
	ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse)
	// GetInventoryForAdjustment returns the inventory record for a SKU+location pair.
	// Used by AdjustmentsService to validate available_qty before decrease/count_reconcile.
	GetInventoryForAdjustment(sku, location string) (*database.Inventory, *responses.InternalResponse)
}
