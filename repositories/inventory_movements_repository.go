package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type InventoryMovementsRepository struct {
	DB *gorm.DB
}

func (r *InventoryMovementsRepository) GetAllInventoryMovements() ([]responses.InventoryMovementView, *responses.InternalResponse) {
	var movements []responses.InventoryMovementView

	sqlRar := `
		SELECT
			im.sku,
			ar.description,
			im."location",
			im.movement_type,
			im.quantity,
			im.remaining_stock,
			im.reason,
			usr.first_name || ' ' || usr.last_name AS created_by,
			im.created_at
		FROM
			inventory_movements im
		INNER JOIN articles ar ON im.sku = ar.sku
		INNER JOIN users usr ON im.created_by = usr.id
	`

	err := r.DB.Raw(sqlRar).Scan(&movements).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los movimientos de inventario",
			Handled: false,
		}
	}

	return movements, nil
}

func (r *InventoryMovementsRepository) GetMovementsBySku(sku string) ([]responses.InventoryMovementView, *responses.InternalResponse) {
	var movements []responses.InventoryMovementView

	sqlRar := `
			SELECT
			im.sku,
			ar.description,
			im."location",
			im.movement_type,
			im.quantity,
			im.remaining_stock,
			im.reason,
			usr.first_name || ' ' || usr.last_name AS created_by,
			im.created_at
		FROM
			inventory_movements im
		INNER JOIN articles ar ON im.sku = ar.sku
		INNER JOIN users usr ON im.created_by = usr.id
		WHERE
			im.sku = ?
	`

	err := r.DB.Raw(sqlRar, sku).Scan(&movements).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los movimientos de inventario",
			Handled: false,
		}
	}

	return movements, nil
}
