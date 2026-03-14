package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// PresentationConversionsRepository defines persistence for presentation conversion rules (WMS convert mode).
type PresentationConversionsRepository interface {
	ListPresentationConversions() ([]database.PresentationConversion, *responses.InternalResponse)
	ListPresentationConversionsAdmin() ([]database.PresentationConversion, *responses.InternalResponse)
	GetPresentationConversionByID(id string) (*database.PresentationConversion, *responses.InternalResponse)
	GetPresentationConversionByFromAndTo(fromID, toID string) (*database.PresentationConversion, *responses.InternalResponse)
	CreatePresentationConversion(req *requests.PresentationConversionCreate) (*database.PresentationConversion, *responses.InternalResponse)
	UpdatePresentationConversion(id string, req *requests.PresentationConversionUpdate) (*database.PresentationConversion, *responses.InternalResponse)
	DeletePresentationConversion(id string) *responses.InternalResponse
}
