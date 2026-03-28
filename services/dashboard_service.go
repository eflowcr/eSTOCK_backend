package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type DashboardService struct {
	Repository ports.DashboardRepository
}

func NewDashboardService(repository ports.DashboardRepository) *DashboardService {
	return &DashboardService{
		Repository: repository,
	}
}

func (s *DashboardService) GetDashboardStats(tasksPeriod string, lowStockThreshold int) (map[string]interface{}, *responses.InternalResponse) {
	return s.Repository.GetDashboardStats(tasksPeriod, lowStockThreshold)
}

func (s *DashboardService) GetInventorySummary(period string) (map[string]interface{}, *responses.InternalResponse) {
	return s.Repository.GetInventorySummary(period)
}

func (s *DashboardService) GetMovementsMonthly(period string) (map[string]interface{}, *responses.InternalResponse) {
	return s.Repository.GetMovementsMonthly(period)
}

func (s *DashboardService) GetRecentActivity() (map[string]interface{}, *responses.InternalResponse) {
	return s.Repository.GetRecentActivity()
}
