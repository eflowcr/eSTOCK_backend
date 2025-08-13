package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type StockAlertsService struct {
	Repository *repositories.StockAlertsRepository
}

func NewStockAlertsService(repo *repositories.StockAlertsRepository) *StockAlertsService {
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
