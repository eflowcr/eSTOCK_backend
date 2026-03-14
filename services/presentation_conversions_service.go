package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type PresentationConversionsService struct {
	Repository ports.PresentationConversionsRepository
}

func NewPresentationConversionsService(repo ports.PresentationConversionsRepository) *PresentationConversionsService {
	return &PresentationConversionsService{Repository: repo}
}

func (s *PresentationConversionsService) ListPresentationConversions() ([]database.PresentationConversion, *responses.InternalResponse) {
	return s.Repository.ListPresentationConversions()
}

func (s *PresentationConversionsService) ListPresentationConversionsAdmin() ([]database.PresentationConversion, *responses.InternalResponse) {
	return s.Repository.ListPresentationConversionsAdmin()
}

func (s *PresentationConversionsService) GetPresentationConversionByID(id string) (*database.PresentationConversion, *responses.InternalResponse) {
	return s.Repository.GetPresentationConversionByID(id)
}

func (s *PresentationConversionsService) GetPresentationConversionByFromAndTo(fromID, toID string) (*database.PresentationConversion, *responses.InternalResponse) {
	return s.Repository.GetPresentationConversionByFromAndTo(fromID, toID)
}

func (s *PresentationConversionsService) CreatePresentationConversion(req *requests.PresentationConversionCreate) (*database.PresentationConversion, *responses.InternalResponse) {
	return s.Repository.CreatePresentationConversion(req)
}

func (s *PresentationConversionsService) UpdatePresentationConversion(id string, req *requests.PresentationConversionUpdate) (*database.PresentationConversion, *responses.InternalResponse) {
	return s.Repository.UpdatePresentationConversion(id, req)
}

func (s *PresentationConversionsService) DeletePresentationConversion(id string) *responses.InternalResponse {
	return s.Repository.DeletePresentationConversion(id)
}
