package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// LocationsService is a thin pass-through to the tenant-aware repository.
// S3.5 W2-A: every method now requires tenantID.
type LocationsService struct {
	Repository ports.LocationsRepository
}

func NewLocationsService(repo ports.LocationsRepository) *LocationsService {
	return &LocationsService{
		Repository: repo,
	}
}

func (s *LocationsService) GetAllLocations(tenantID string) ([]database.Location, *responses.InternalResponse) {
	return s.Repository.GetAllLocations(tenantID)
}

func (s *LocationsService) GetLocationByID(tenantID, id string) (*database.Location, *responses.InternalResponse) {
	return s.Repository.GetLocationByID(tenantID, id)
}

func (s *LocationsService) CreateLocation(tenantID string, loc *requests.Location) *responses.InternalResponse {
	return s.Repository.CreateLocation(tenantID, loc)
}

func (s *LocationsService) UpdateLocation(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateLocation(tenantID, id, data)
}

func (s *LocationsService) DeleteLocation(tenantID, id string) *responses.InternalResponse {
	return s.Repository.DeleteLocation(tenantID, id)
}

func (s *LocationsService) ImportLocationsFromExcel(tenantID string, fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	return s.Repository.ImportLocationsFromExcel(tenantID, fileBytes)
}

func (s *LocationsService) ImportLocationsFromJSON(tenantID string, rows []requests.LocationImportRow) ([]string, []string, *responses.InternalResponse) {
	return s.Repository.ImportLocationsFromJSON(tenantID, rows)
}

func (s *LocationsService) ValidateImportRows(tenantID string, rows []requests.LocationImportRow) ([]responses.LocationValidationResult, *responses.InternalResponse) {
	return s.Repository.ValidateImportRows(tenantID, rows)
}

func (s *LocationsService) ExportLocationsToExcel(tenantID string) ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportLocationsToExcel(tenantID)
}

func (s *LocationsService) GenerateImportTemplate(language string) ([]byte, error) {
	return s.Repository.GenerateImportTemplate(language)
}
