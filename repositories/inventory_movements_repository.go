package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type InventoryMovementsRepository struct {
	DB *gorm.DB
}

func (r *InventoryMovementsRepository) GetAllInventoryMovements(sku string) ([]database.InventoryMovement, *responses.InternalResponse) {
	var movements []database.InventoryMovement

	err := r.DB.
		Table(database.InventoryMovement{}.TableName()).
		Where("sku = ?", sku).
		Order("created_at DESC").
		Find(&movements).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch inventory movements",
			Handled: false,
		}
	}

	return movements, nil
}
