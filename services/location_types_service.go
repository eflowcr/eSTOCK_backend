package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type LocationTypesService struct {
	Repository ports.LocationTypesRepository
}

func NewLocationTypesService(repo ports.LocationTypesRepository) *LocationTypesService {
	return &LocationTypesService{Repository: repo}
}

func (s *LocationTypesService) ListLocationTypes() ([]database.LocationType, *responses.InternalResponse) {
	return s.Repository.ListLocationTypes()
}

func (s *LocationTypesService) ListLocationTypesAdmin() ([]database.LocationType, *responses.InternalResponse) {
	return s.Repository.ListLocationTypesAdmin()
}

func (s *LocationTypesService) GetLocationTypeByID(id string) (*database.LocationType, *responses.InternalResponse) {
	return s.Repository.GetLocationTypeByID(id)
}

func (s *LocationTypesService) GetLocationTypeByCode(code string) (*database.LocationType, *responses.InternalResponse) {
	return s.Repository.GetLocationTypeByCode(code)
}

func (s *LocationTypesService) CreateLocationType(req *requests.LocationTypeCreate) (*database.LocationType, *responses.InternalResponse) {
	return s.Repository.CreateLocationType(req)
}

func (s *LocationTypesService) UpdateLocationType(id string, req *requests.LocationTypeUpdate) (*database.LocationType, *responses.InternalResponse) {
	return s.Repository.UpdateLocationType(id, req)
}

func (s *LocationTypesService) DeleteLocationType(id string) *responses.InternalResponse {
	return s.Repository.DeleteLocationType(id)
}
