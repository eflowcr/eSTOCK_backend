package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type StockAlertsRepository struct {
	DB *gorm.DB
}

func (r *StockAlertsRepository) GetAllStockAlerts(resolved bool) ([]database.StockAlert, *responses.InternalResponse) {
	var stockAlerts []database.StockAlert

	err := r.DB.
		Table(database.StockAlert{}.TableName()).
		Where("is_resolved = ?", resolved).
		Order("created_at ASC").
		Find(&stockAlerts).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch stock alerts",
			Handled: false,
		}
	}

	if len(stockAlerts) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No stock alerts found",
			Handled: true,
		}
	}

	return stockAlerts, nil
}
