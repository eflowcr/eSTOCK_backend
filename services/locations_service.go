package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type LocationsService struct {
	Repository *repositories.LocationsRepository
}

func NewLocationsService(repo *repositories.LocationsRepository) *LocationsService {
	return &LocationsService{
		Repository: repo,
	}
}

func (s *LocationsService) GetAllLocations() ([]database.Location, *responses.InternalResponse) {
	return s.Repository.GetAllLocations()
}

func (s *LocationsService) GetLocationByID(id string) (*database.Location, *responses.InternalResponse) {
	return s.Repository.GetLocationByID(id)
}

func (s *LocationsService) CreateLocation(loc *requests.Location) *responses.InternalResponse {
	return s.Repository.CreateLocation(loc)
}

func (s *LocationsService) UpdateLocation(id int, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateLocation(id, data)
}

func (s *LocationsService) DeleteLocation(id int) *responses.InternalResponse {
	return s.Repository.DeleteLocation(id)
}

func (s *LocationsService) ImportLocationsFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return s.Repository.ImportLocationsFromExcel(fileBytes)
}

func (s *LocationsService) ExportLocationsToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportLocationsToExcel()
}
