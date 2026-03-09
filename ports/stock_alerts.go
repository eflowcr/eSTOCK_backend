package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// StockAlertsRepository defines persistence operations for stock alerts.
type StockAlertsRepository interface {
	GetAllStockAlerts(resolved bool) ([]database.StockAlert, *responses.InternalResponse)
	Analyze() (*responses.StockAlertResponse, *responses.InternalResponse)
	LotExpiration() (*responses.StockAlertResponse, *responses.InternalResponse)
	ResolveAlert(alertID string) *responses.InternalResponse
	ExportAlertsToExcel() ([]byte, *responses.InternalResponse)
}
