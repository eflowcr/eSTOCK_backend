package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
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

func (s *GamificationService) Badges(userId string) ([]database.Badge, *responses.InternalResponse) {
	return s.Repository.Badges(userId)
}

func (s *GamificationService) GetAllBadges() ([]database.Badge, *responses.InternalResponse) {
	return s.Repository.GetAllBadges()
}

func (s *GamificationService) CompleteTasks(userId string, task requests.CompleteTasks) ([]database.UserBadge, *responses.InternalResponse) {
	return s.Repository.CompleteTasks(userId, task)
}
