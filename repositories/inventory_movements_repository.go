package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
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
			Message: "Error al obtener los movimientos de inventario",
			Handled: false,
		}
	}

	return movements, nil
}

func (r *InventoryMovementsRepository) ListMovements(f ports.MovementsFilter) ([]database.InventoryMovement, *responses.InternalResponse) {
	var movements []database.InventoryMovement
	q := r.DB.Table(database.InventoryMovement{}.TableName()).Order("created_at DESC")

	if f.SKU != "" {
		q = q.Where("sku = ?", f.SKU)
	}
	if f.Location != "" {
		q = q.Where("location = ?", f.Location)
	}
	if f.LotID != "" {
		q = q.Where("lot_id = ?", f.LotID)
	}
	if f.MovementType != "" {
		q = q.Where("movement_type = ?", f.MovementType)
	}
	if f.ReferenceType != "" {
		q = q.Where("reference_type = ?", f.ReferenceType)
	}
	if f.UserID != "" {
		q = q.Where("user_id = ?", f.UserID)
	}
	if f.From != "" {
		q = q.Where("created_at >= ?", f.From)
	}
	if f.To != "" {
		q = q.Where("created_at <= ?", f.To+" 23:59:59")
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	} else {
		q = q.Limit(500)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}

	if err := q.Find(&movements).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al listar movimientos de inventario",
			Handled: false,
		}
	}
	return movements, nil
}

// ListRecentMovements returns the newest movements across the whole tenant.
//
// inventory_movements has no tenant_id column (migration 000019 only added it to
// picking_tasks/receiving_tasks/adjustments; movements were not retrofitted). To
// keep this mobile-only feed tenant-isolated we join against articles, which IS
// tenant-scoped (migration 000029, unique on (tenant_id, sku)). A movement whose
// sku does not belong to the requesting tenant is excluded. When tenantID is
// empty (single-tenant / test fallback) we skip the join and return movements
// directly so behaviour matches the legacy single-tenant deployments.
func (r *InventoryMovementsRepository) ListRecentMovements(tenantID string, limit int) ([]database.InventoryMovement, *responses.InternalResponse) {
	var movements []database.InventoryMovement

	if limit <= 0 {
		limit = 30
	}
	if limit > 200 {
		limit = 200
	}

	q := r.DB.
		Table("inventory_movements AS m").
		Select("m.*").
		Order("m.created_at DESC").
		Limit(limit)

	if tenantID != "" {
		q = q.
			Joins("JOIN articles a ON a.sku = m.sku").
			Where("a.tenant_id = ?", tenantID)
	}

	if err := q.Find(&movements).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener los movimientos recientes",
			Handled: false,
		}
	}

	return movements, nil
}
