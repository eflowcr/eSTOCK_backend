package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

type SerialsRepository struct {
	DB *gorm.DB
}

func (r *SerialsRepository) GetSerialByID(id int) (*database.Serial, *responses.InternalResponse) {
	var serial database.Serial

	err := r.DB.Table(database.Serial{}.TableName()).
		Where("id = ?", id).
		First(&serial).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch serial",
			Handled: false,
		}
	}

	return &serial, nil
}

func (r *SerialsRepository) GetSerialsBySKU(sku string) ([]database.Serial, *responses.InternalResponse) {
	var serials []database.Serial

	err := r.DB.Table(database.Serial{}.TableName()).
		Where("sku = ?", sku).
		Order("created_at DESC").
		Find(&serials).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch serials",
			Handled: false,
		}
	}

	return serials, nil
}

func (r *SerialsRepository) CreateSerial(data *requests.CreateSerialRequest) *responses.InternalResponse {
	now := tools.GetCurrentTime()

	serial := &database.Serial{
		SerialNumber: data.SerialNumber,
		SKU:          data.SKU,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := r.DB.Create(serial).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to create serial",
			Handled: false,
		}
	}

	return nil
}
