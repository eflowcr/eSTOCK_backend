package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type StockAlertsService struct {
	Repository ports.StockAlertsRepository
}

func NewStockAlertsService(repo ports.StockAlertsRepository) *StockAlertsService {
	return &StockAlertsService{
		Repository: repo,
	}
}

// S3.5 W2-B: every method threads tenantID. Cron callers must iterate the active tenants
// list and invoke Analyze/LotExpiration per tenant; HTTP callers pass Config.TenantID.

func (s *StockAlertsService) GetAllStockAlerts(tenantID string, resolved bool) ([]database.StockAlert, *responses.InternalResponse) {
	return s.Repository.GetAllStockAlerts(tenantID, resolved)
}

func (s *StockAlertsService) Analyze(tenantID string) (*responses.StockAlertResponse, *responses.InternalResponse) {
	return s.Repository.Analyze(tenantID)
}

func (s *StockAlertsService) LotExpiration(tenantID string) (*responses.StockAlertResponse, *responses.InternalResponse) {
	return s.Repository.LotExpiration(tenantID)
}

func (s *StockAlertsService) ResolveAlert(tenantID, alertID string) *responses.InternalResponse {
	return s.Repository.ResolveAlert(tenantID, alertID)
}

func (s *StockAlertsService) ExportAlertsToExcel(tenantID string) ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportAlertsToExcel(tenantID)
}
