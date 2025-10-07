package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type InventoryMovementsRepository struct {
	DB *gorm.DB
}

func (r *InventoryMovementsRepository) GetAllInventoryMovements() ([]database.InventoryMovement, *responses.InternalResponse) {
	var movements []database.InventoryMovement

	err := r.DB.
		Table("inventory_movements").
		Order("created_at DESC").
		Find(&movements).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los movimientos de inventario",
			Handled: false,
		}
	}

	return movements, nil
}

func (r *InventoryMovementsRepository) GetMovementsBySku(sku string) ([]database.InventoryMovement, *responses.InternalResponse) {
	var movements []database.InventoryMovement

	err := r.DB.
		Table("inventory_movements").
		Where("sku = ?", sku).
		Order("created_at DESC").
		Find(&movements).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los movimientos de inventario",
			Handled: false,
		}
	}

	return movements, nil
}
