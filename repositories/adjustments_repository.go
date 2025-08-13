package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type AdjustmentsRepository struct {
	DB *gorm.DB
}

func (r *AdjustmentsRepository) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	var adjustments []database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Order("created_at ASC").
		Find(&adjustments).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch adjustments",
			Handled: false,
		}
	}

	if len(adjustments) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No adjustments found",
			Handled: true,
		}
	}

	return adjustments, nil
}

func (r *AdjustmentsRepository) GetAdjustmentByID(id int) (*database.Adjustment, *responses.InternalResponse) {
	var adjustment database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Where("id = ?", id).
		First(&adjustment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Adjustment not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch adjustment",
			Handled: false,
		}
	}

	return &adjustment, nil
}

func (r *AdjustmentsRepository) GetAdjustmentDetails(id int) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	var adjustment database.Adjustment

	err := r.DB.
		Table(database.Adjustment{}.TableName()).
		Where("id = ?", id).
		First(&adjustment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Adjustment not found",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch adjustment details",
			Handled: false,
		}
	}

	// Get inventory
	var inventory database.Inventory

	err = r.DB.
		Table(database.Inventory{}.TableName()).
		Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
		First(&inventory).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Inventory not found for this adjustment",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory details",
			Handled: false,
		}
	}

	// Get lots for inventory
	var lots []database.Lot

	err = r.DB.
		Table(database.Lot{}.TableName()).
		Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
		Find(&lots).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch lots for inventory",
			Handled: false,
		}
	}

	// Get serials for inventory
	var serials []database.Serial

	err = r.DB.
		Table(database.Serial{}.TableName()).
		Where("sku = ? AND location = ?", adjustment.SKU, adjustment.Location).
		Find(&serials).Error
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch serials for inventory",
			Handled: false,
		}
	}

	// Get article
	var article database.Article
	err = r.DB.
		Table(database.Article{}.TableName()).
		Where("sku = ?", adjustment.SKU).
		First(&article).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   nil,
				Message: "Article not found for this adjustment",
				Handled: true,
			}
		}
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch article details",
			Handled: false,
		}
	}

	details := dto.AdjustmentDetails{
		Adjustment: adjustment,
		Inventory:  inventory,
		Lots:       lots,
		Serials:    serials,
		Article:    article,
	}

	return &details, nil
}
