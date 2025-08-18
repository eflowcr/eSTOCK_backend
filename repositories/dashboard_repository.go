package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type DashboardRepository struct {
	DB *gorm.DB
}

func (r *DashboardRepository) GetDashboardStats() (map[string]interface{}, *responses.InternalResponse) {
	var totalSkus int64
	err := r.DB.Table("inventory").Distinct("sku").Count(&totalSkus).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to count SKUs", Handled: false}
	}

	var inventoryValue float64
	err = r.DB.
		Table("inventory").
		Select("SUM(inventory.quantity * COALESCE(articles.unit_price, 0))").
		Joins("LEFT JOIN articles ON inventory.sku = articles.sku").
		Scan(&inventoryValue).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to calculate inventory value", Handled: false}
	}

	var lowStockCount int64
	err = r.DB.
		Table("inventory").
		Where("quantity < ?", 20).
		Count(&lowStockCount).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to count low stock", Handled: false}
	}

	var activeReceiving int64
	err = r.DB.
		Table("receiving_tasks").
		Where("status IN ?", []string{"open", "in_progress"}).
		Count(&activeReceiving).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to count receiving tasks", Handled: false}
	}

	var activePicking int64
	err = r.DB.
		Table("picking_tasks").
		Where("status IN ?", []string{"open", "in_progress"}).
		Count(&activePicking).Error
	if err != nil {
		return nil, &responses.InternalResponse{Error: err, Message: "Failed to count picking tasks", Handled: false}
	}

	result := map[string]interface{}{
		"totalSkus":      totalSkus,
		"inventoryValue": inventoryValue,
		"lowStockCount":  lowStockCount,
		"activeTasks":    activeReceiving + activePicking,
	}

	return result, nil
}
