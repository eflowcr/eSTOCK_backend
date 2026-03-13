package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// PresentationTypesRepository defines persistence for presentation types (admin + dropdown).
type PresentationTypesRepository interface {
	ListPresentationTypes() ([]database.PresentationType, *responses.InternalResponse)
	ListPresentationTypesAdmin() ([]database.PresentationType, *responses.InternalResponse)
	GetPresentationTypeByID(id string) (*database.PresentationType, *responses.InternalResponse)
	GetPresentationTypeByCode(code string) (*database.PresentationType, *responses.InternalResponse)
	CreatePresentationType(req *requests.PresentationTypeCreate) (*database.PresentationType, *responses.InternalResponse)
	UpdatePresentationType(id string, req *requests.PresentationTypeUpdate) (*database.PresentationType, *responses.InternalResponse)
	DeletePresentationType(id string) *responses.InternalResponse
}
