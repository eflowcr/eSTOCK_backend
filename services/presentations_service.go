package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type PresentationsService struct {
	Repository *repositories.PresentationsRepository
}

func NewPresentationsService(repo *repositories.PresentationsRepository) *PresentationsService {
	return &PresentationsService{
		Repository: repo,
	}
}

func (s *PresentationsService) GetAllPresentations() ([]database.Presentations, *responses.InternalResponse) {
	return s.Repository.GetAllPresentations()
}

func (s *PresentationsService) GetPresentationByID(id string) (*database.Presentations, *responses.InternalResponse) {
	return s.Repository.GetPresentationByID(id)
}

func (s *PresentationsService) CreatePresentation(data *database.Presentations) *responses.InternalResponse {
	return s.Repository.CreatePresentation(data)
}

func (s *PresentationsService) UpdatePresentation(id, name string) (*database.Presentations, *responses.InternalResponse) {
	return s.Repository.UpdatePresentation(id, name)
}

func (s *PresentationsService) DeletePresentation(id string) *responses.InternalResponse {
	return s.Repository.DeletePresentation(id)
}
