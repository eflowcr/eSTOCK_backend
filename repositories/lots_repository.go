package repositories

import (
	"errors"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"gorm.io/gorm"
)

type LotsRepository struct {
	DB *gorm.DB
}

func (r *LotsRepository) GetAllLots() ([]database.Lot, *responses.InternalResponse) {
	var lots []database.Lot

	err := r.DB.Table(database.Lot{}.TableName()).Order("created_at DESC").Find(&lots).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch lots",
			Handled: false,
		}
	}

	return lots, nil
}

func (r *LotsRepository) GetLotsBySKU(sku *string) ([]database.Lot, *responses.InternalResponse) {
	var lots []database.Lot

	query := r.DB.Table(database.Lot{}.TableName())

	if sku != nil && *sku != "" {
		query = query.Where("sku = ?", *sku)
	} else {
		query = query.Order("created_at DESC")
	}

	err := query.Find(&lots).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch lots",
			Handled: false,
		}
	}

	return lots, nil
}

func (r *LotsRepository) CreateLot(data *requests.CreateLotRequest) *responses.InternalResponse {
	now := tools.GetCurrentTime()

	lot := &database.Lot{
		LotNumber:      data.LotNumber,
		SKU:            data.SKU,
		Quantity:       data.Quantity,
		ExpirationDate: data.ExpirationDate,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := r.DB.Create(lot).Error
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to create lot",
			Handled: false,
		}
	}

	return nil
}

func (r *LotsRepository) UpdateLot(id int, data map[string]interface{}) *responses.InternalResponse {
	var lot database.Lot

	err := r.DB.First(&lot, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Lot not found",
			Handled: true,
		}
	}
	if err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to retrieve lot",
			Handled: false,
		}
	}

	protectedFields := map[string]bool{
		"id":         true,
		"created_at": true,
	}

	for k := range protectedFields {
		delete(data, k)
	}

	data["updated_at"] = tools.GetCurrentTime()

	if err := r.DB.Table(database.Lot{}.TableName()).Where(
		"id = ?", id,
	).Updates(data).Error; err != nil {
		return &responses.InternalResponse{
			Error:   err,
			Message: "Failed to update lot",
			Handled: false,
		}
	}

	return nil
}

func (r *LotsRepository) DeleteLot(id int) *responses.InternalResponse {
	result := r.DB.Delete(&database.Lot{}, id)
	if result.Error != nil {
		return &responses.InternalResponse{
			Error:   result.Error,
			Message: "Failed to delete lot",
			Handled: false,
		}
	}

	if result.RowsAffected == 0 {
		return &responses.InternalResponse{
			Error:   nil,
			Message: "Lot not found",
			Handled: true,
		}
	}

	return nil
}
