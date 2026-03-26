package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type LocationsService struct {
	Repository ports.LocationsRepository
}

func NewLocationsService(repo ports.LocationsRepository) *LocationsService {
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

func (s *LocationsService) UpdateLocation(id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateLocation(id, data)
}

func (s *LocationsService) DeleteLocation(id string) *responses.InternalResponse {
	return s.Repository.DeleteLocation(id)
}

func (s *LocationsService) ImportLocationsFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return s.Repository.ImportLocationsFromExcel(fileBytes)
}

func (s *LocationsService) ExportLocationsToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportLocationsToExcel()
}

func (s *LocationsService) GenerateImportTemplate(language string) ([]byte, error) {
	return s.Repository.GenerateImportTemplate(language)
}
