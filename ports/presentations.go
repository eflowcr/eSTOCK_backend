package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// PresentationsRepository defines persistence operations for presentations.
type PresentationsRepository interface {
	GetAllPresentations() ([]database.Presentations, *responses.InternalResponse)
	GetPresentationByID(id string) (*database.Presentations, *responses.InternalResponse)
	CreatePresentation(data *database.Presentations) *responses.InternalResponse
	UpdatePresentation(id, name string) (*database.Presentations, *responses.InternalResponse)
	DeletePresentation(id string) *responses.InternalResponse
}
