package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type GamificationRepository struct {
	DB *gorm.DB
}

func (r *GamificationRepository) GamificationStats(userId string) (*database.UserStat, *responses.InternalResponse) {
	// Get user stats from the database
	var userStat database.UserStat
	if err := r.DB.Where("user_id = ?", userId).First(&userStat).Error; err != nil {
		// If record not found, stats would be created next, so ignore the error

		if err != gorm.ErrRecordNotFound {
			return nil, &responses.InternalResponse{
				Error:   err,
				Message: "Failed to fetch user stats",
				Handled: false,
			}
		}
	}

	// If user stats do not exist, create a new record with default values
	if userStat.ID == 0 {
		userStat = database.UserStat{
			UserID:                  userId,
			ReceivingTasksCompleted: 0,
			PickingTasksCompleted:   0,
			PickAccuracy:            100,
			AvgPickTime:             0,
		}

		if err := r.DB.Create(&userStat).Error; err != nil {
			return nil, &responses.InternalResponse{
				Error:   err,
				Message: "Failed to create user stats",
				Handled: false,
			}
		}
	}

	return &userStat, nil
}
