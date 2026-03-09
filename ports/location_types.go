package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// LocationTypesRepository defines persistence for location types (admin + dropdown).
type LocationTypesRepository interface {
	ListLocationTypes() ([]database.LocationType, *responses.InternalResponse)
	ListLocationTypesAdmin() ([]database.LocationType, *responses.InternalResponse)
	GetLocationTypeByID(id string) (*database.LocationType, *responses.InternalResponse)
	GetLocationTypeByCode(code string) (*database.LocationType, *responses.InternalResponse)
	CreateLocationType(req *requests.LocationTypeCreate) (*database.LocationType, *responses.InternalResponse)
	UpdateLocationType(id string, req *requests.LocationTypeUpdate) (*database.LocationType, *responses.InternalResponse)
	DeleteLocationType(id string) *responses.InternalResponse
}
