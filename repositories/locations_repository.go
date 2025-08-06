package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type LocationsRepository struct {
	DB *gorm.DB
}

func (r *LocationsRepository) GetAllLocations() ([]database.Location, *responses.InternalResponse) {
	var locations []database.Location

	err := r.DB.
		Table(database.Location{}.TableName()).
		Order("created_at ASC").
		Find(&locations).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch locations",
			Handled: false,
		}
	}

	if len(locations) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No locations found",
			Handled: true,
		}
	}

	return locations, nil
}
