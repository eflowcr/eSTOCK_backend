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

func (s *DashboardService) GetDashboardStats() (map[string]interface{}, *responses.InternalResponse) {
	return s.Repository.GetDashboardStats()
}
