package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type PresentationTypesService struct {
	Repository ports.PresentationTypesRepository
}

func NewPresentationTypesService(repo ports.PresentationTypesRepository) *PresentationTypesService {
	return &PresentationTypesService{Repository: repo}
}

func (s *PresentationTypesService) ListPresentationTypes() ([]database.PresentationType, *responses.InternalResponse) {
	return s.Repository.ListPresentationTypes()
}

func (s *PresentationTypesService) ListPresentationTypesAdmin() ([]database.PresentationType, *responses.InternalResponse) {
	return s.Repository.ListPresentationTypesAdmin()
}

func (s *PresentationTypesService) GetPresentationTypeByID(id string) (*database.PresentationType, *responses.InternalResponse) {
	return s.Repository.GetPresentationTypeByID(id)
}

func (s *PresentationTypesService) GetPresentationTypeByCode(code string) (*database.PresentationType, *responses.InternalResponse) {
	return s.Repository.GetPresentationTypeByCode(code)
}

func (s *PresentationTypesService) CreatePresentationType(req *requests.PresentationTypeCreate) (*database.PresentationType, *responses.InternalResponse) {
	return s.Repository.CreatePresentationType(req)
}

func (s *PresentationTypesService) UpdatePresentationType(id string, req *requests.PresentationTypeUpdate) (*database.PresentationType, *responses.InternalResponse) {
	return s.Repository.UpdatePresentationType(id, req)
}

func (s *PresentationTypesService) DeletePresentationType(id string) *responses.InternalResponse {
	return s.Repository.DeletePresentationType(id)
}
