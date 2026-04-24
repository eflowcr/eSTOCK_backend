package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// LotsRepository defines persistence operations for lots.
//
// S3.5 W2-B: every public method takes a tenantID so multi-tenant deployments cannot
// cross-leak lot data via SKU lookup or list/trace endpoints. Internal lookups that
// already validated tenancy via a parent record (e.g. picking task → lot via
// inventory_movements) keep using GetLotByID without a tenant filter.
type LotsRepository interface {
	GetAllLots(tenantID string) ([]database.Lot, *responses.InternalResponse)
	GetLotsBySKU(tenantID string, sku *string) ([]database.Lot, *responses.InternalResponse)
	GetLotByID(id string) (*database.Lot, *responses.InternalResponse)
	GetLotByIDForTenant(id, tenantID string) (*database.Lot, *responses.InternalResponse)
	CreateLot(tenantID string, data *requests.CreateLotRequest) *responses.InternalResponse
	UpdateLot(tenantID, id string, data map[string]interface{}) *responses.InternalResponse
	DeleteLot(tenantID, id string) *responses.InternalResponse
	GetLotTrace(tenantID, lotID string) (*responses.LotTraceResponse, *responses.InternalResponse)
}
