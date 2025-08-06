package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
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
