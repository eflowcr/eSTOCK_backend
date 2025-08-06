package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type DashboardService struct {
	Repository *repositories.DashboardRepository
}

func NewDashboardService(repository *repositories.DashboardRepository) *DashboardService {
	return &DashboardService{
		Repository: repository,
	}
}

func (s *DashboardService) GetDashboardStats() (map[string]interface{}, *responses.InternalResponse) {
	return s.Repository.GetDashboardStats()
}
