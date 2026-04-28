package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// StockAlertsRepository defines persistence operations for stock alerts.
//
// S3.5 W2-B: every method is tenant-scoped. Analyze() reads inventory and lots, both of
// which are tenant-scoped (S2.5 + S3.5 W2-B); without a tenantID parameter the
// truncate/recompute would mix data across tenants and overwrite each tenant's alerts
// every time another tenant ran the analyzer.
type StockAlertsRepository interface {
	GetAllStockAlerts(tenantID string, resolved bool) ([]database.StockAlert, *responses.InternalResponse)
	Analyze(tenantID string) (*responses.StockAlertResponse, *responses.InternalResponse)
	LotExpiration(tenantID string) (*responses.StockAlertResponse, *responses.InternalResponse)
	ResolveAlert(tenantID, alertID string) *responses.InternalResponse
	ExportAlertsToExcel(tenantID string) ([]byte, *responses.InternalResponse)
}
