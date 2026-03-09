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

func (s *StockAlertsService) GetAllStockAlerts(resolved bool) ([]database.StockAlert, *responses.InternalResponse) {
	return s.Repository.GetAllStockAlerts(resolved)
}

func (s *StockAlertsService) Analyze() (*responses.StockAlertResponse, *responses.InternalResponse) {
	return s.Repository.Analyze()
}

func (s *StockAlertsService) LotExpiration() (*responses.StockAlertResponse, *responses.InternalResponse) {
	return s.Repository.LotExpiration()
}

func (s *StockAlertsService) ResolveAlert(alertID string) *responses.InternalResponse {
	return s.Repository.ResolveAlert(alertID)
}

func (s *StockAlertsService) ExportAlertsToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportAlertsToExcel()
}
