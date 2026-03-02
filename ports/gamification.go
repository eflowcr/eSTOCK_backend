package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// GamificationRepository defines persistence operations for gamification.
type GamificationRepository interface {
	GamificationStats(userId string) (*database.UserStat, *responses.InternalResponse)
	Badges(userId string) ([]database.Badge, *responses.InternalResponse)
	GetAllBadges() ([]database.Badge, *responses.InternalResponse)
	CompleteTasks(userId string, task requests.CompleteTasks) ([]database.UserBadge, *responses.InternalResponse)
	GetAllStats() ([]responses.UserStatsResponse, *responses.InternalResponse)
}
