package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// LocationsRepository defines persistence operations for locations.
type LocationsRepository interface {
	GetAllLocations() ([]database.Location, *responses.InternalResponse)
	GetLocationByID(id string) (*database.Location, *responses.InternalResponse)
	CreateLocation(loc *requests.Location) *responses.InternalResponse
	UpdateLocation(id string, data map[string]interface{}) *responses.InternalResponse
	DeleteLocation(id string) *responses.InternalResponse
	ImportLocationsFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse)
	ExportLocationsToExcel() ([]byte, *responses.InternalResponse)
	GenerateImportTemplate(language string) ([]byte, error)
}
