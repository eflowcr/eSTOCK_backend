package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type GamificationService struct {
	Repository *repositories.GamificationRepository
}

func NewGamificationService(repo *repositories.GamificationRepository) *GamificationService {
	return &GamificationService{
		Repository: repo,
	}
}

func (s *GamificationService) GamificationStats(userId string) (*database.UserStat, *responses.InternalResponse) {
	return s.Repository.GamificationStats(userId)
}
