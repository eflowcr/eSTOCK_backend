package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// LocationsRepository defines persistence operations for locations.
//
// S3.5 W2-A: all methods are tenant-scoped. Cross-tenant lookups (used only by
// internal admin/scan flows that are not part of W2-A) are not exposed here.
type LocationsRepository interface {
	GetAllLocations(tenantID string) ([]database.Location, *responses.InternalResponse)
	GetLocationByID(tenantID, id string) (*database.Location, *responses.InternalResponse)
	CreateLocation(tenantID string, loc *requests.Location) *responses.InternalResponse
	UpdateLocation(tenantID, id string, data map[string]interface{}) *responses.InternalResponse
	DeleteLocation(tenantID, id string) *responses.InternalResponse
	ImportLocationsFromExcel(tenantID string, fileBytes []byte) ([]string, []string, *responses.InternalResponse)
	ImportLocationsFromJSON(tenantID string, rows []requests.LocationImportRow) ([]string, []string, *responses.InternalResponse)
	ValidateImportRows(tenantID string, rows []requests.LocationImportRow) ([]responses.LocationValidationResult, *responses.InternalResponse)
	ExportLocationsToExcel(tenantID string) ([]byte, *responses.InternalResponse)
	GenerateImportTemplate(language string) ([]byte, error)
}
