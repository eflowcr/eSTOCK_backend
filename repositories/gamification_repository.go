package repositories

import (
	"encoding/json"
	"math"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type BadgeCriteria struct {
	PickingTasks   *int     `json:"pickingTasks"`
	ReceivingTasks *int     `json:"receivingTasks"`
	Accuracy       *float64 `json:"accuracy"`
	AvgTime        *float64 `json:"avgTime"`
	TotalTasks     *int     `json:"totalTasks"`
}

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
				Message: "Error al obtener las estadísticas del usuario",
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
				Message: "Error al crear las estadísticas del usuario",
				Handled: false,
			}
		}
	}

	return &userStat, nil
}

func (r *GamificationRepository) Badges(userId string) ([]database.Badge, *responses.InternalResponse) {
	var badges []database.Badge
	if err := r.DB.Where("user_id = ?", userId).Find(&badges).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener las insignias",
			Handled: false,
		}
	}

	return badges, nil
}

func (r *GamificationRepository) GetAllBadges() ([]database.Badge, *responses.InternalResponse) {
	var badges []database.Badge
	if err := r.DB.Find(&badges).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener todas las insignias",
			Handled: false,
		}
	}

	return badges, nil
}

func (r *GamificationRepository) CompleteTasks(userId string, task requests.CompleteTasks) ([]database.UserBadge, *responses.InternalResponse) {
	if task.CompletionTime > 3600 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "El tiempo de finalización de la tarea excede 1 hora",
			Handled: true,
		}
	}

	if task.TaskType == "picking" && task.Accuracy == nil {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "Se requiere precisión para las tareas de picking",
			Handled: true,
		}
	}

	// Get user stats
	userStat, errResp := r.GamificationStats(userId)

	if errResp != nil {
		return nil, errResp
	}

	var updates database.UserStat

	switch task.TaskType {
	case "receiving":
		updates.ReceivingTasksCompleted = userStat.ReceivingTasksCompleted + 1
	case "picking":
		updates.PickingTasksCompleted = userStat.PickingTasksCompleted + 1
		updates.TotalPickingTime = userStat.TotalPickingTime + task.CompletionTime
		updates.AvgPickTime = (userStat.TotalPickingTime + task.CompletionTime) / updates.PickingTasksCompleted

		if task.Accuracy != nil {
			newTotalPicks := userStat.TotalPicks + 1
			value := 0

			if *task.Accuracy == 100 {
				value = 1
			}

			newCorrectPicks := userStat.CorrectPicks + value
			updates.TotalPicks = newTotalPicks
			updates.CorrectPicks = newCorrectPicks
			updates.PickAccuracy = int(math.Round((float64(newCorrectPicks) / float64(newTotalPicks)) * 100))
		}
	}

	// Update user stats in the database
	if err := r.DB.Model(&database.UserStat{}).Where("user_id = ?", userId).Updates(updates).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al actualizar las estadísticas del usuario",
			Handled: false,
		}
	}

	return r.CheckAndAwardBadges(userId)
}

func (s *GamificationRepository) CheckAndAwardBadges(userID string) ([]database.UserBadge, *responses.InternalResponse) {
	stats, err := s.GamificationStats(userID)
	if err != nil || stats == nil {
		return nil, err
	}

	allBadges, err := s.GetAllBadges()
	if err != nil {
		return nil, err
	}

	userBadges, err := s.Badges(userID)
	if err != nil {
		return nil, err
	}

	userBadgeIDs := make(map[int]bool, len(userBadges))
	for _, ub := range userBadges {
		userBadgeIDs[ub.ID] = true
	}

	var newBadges []database.UserBadge

	for _, badge := range allBadges {
		if userBadgeIDs[badge.ID] {
			continue
		}

		var criteria BadgeCriteria
		if err := json.Unmarshal(badge.Criteria, &criteria); err != nil {
			// si falla el parseo, ignoramos esta insignia
			continue
		}

		var shouldAward bool

		switch badge.RuleType {
		case "perfect_picker":
			if criteria.PickingTasks != nil && criteria.Accuracy != nil {
				shouldAward =
					stats.PickingTasksCompleted >= *criteria.PickingTasks &&
						stats.PickAccuracy >= int(*criteria.Accuracy)
			}
		case "quick_receiver":
			if criteria.ReceivingTasks != nil && criteria.AvgTime != nil {
				avgReceivingTime := 0.0
				if stats.ReceivingTasksCompleted > 0 {
					avgReceivingTime = float64(stats.TotalPickingTime) / float64(stats.ReceivingTasksCompleted) / 60.0
				}
				shouldAward =
					stats.ReceivingTasksCompleted >= *criteria.ReceivingTasks &&
						avgReceivingTime <= *criteria.AvgTime
			}
		case "speed_demon":
			if criteria.PickingTasks != nil && criteria.AvgTime != nil {
				avgPickTime := 0.0
				if stats.PickingTasksCompleted > 0 {
					avgPickTime = float64(stats.TotalPickingTime) / float64(stats.PickingTasksCompleted) / 60.0
				}
				shouldAward =
					stats.PickingTasksCompleted >= *criteria.PickingTasks &&
						avgPickTime <= *criteria.AvgTime
			}
		case "accuracy_master":
			if criteria.TotalTasks != nil && criteria.Accuracy != nil {
				totalTasks := stats.PickingTasksCompleted + stats.ReceivingTasksCompleted
				shouldAward =
					totalTasks >= *criteria.TotalTasks &&
						stats.PickAccuracy >= int(*criteria.Accuracy)
			}
		case "task_champion":
			if criteria.TotalTasks != nil {
				totalTasks := stats.PickingTasksCompleted + stats.ReceivingTasksCompleted
				shouldAward = totalTasks >= *criteria.TotalTasks
			}
		default:
			continue
		}

		if shouldAward {
			newBadge, err := s.AwardBadge(userID, badge.ID)
			if err != nil {
				continue
			}
			newBadges = append(newBadges, *newBadge)
		}
	}

	return newBadges, nil
}

func (r *GamificationRepository) AwardBadge(userId string, badgeId int) (*database.UserBadge, *responses.InternalResponse) {
	var userBadge database.UserBadge
	userBadge.UserID = userId
	userBadge.BadgeID = badgeId

	if err := r.DB.Create(&userBadge).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to award badge",
			Handled: false,
		}
	}

	return &userBadge, nil
}

func (r *GamificationRepository) GetAllStats() ([]responses.UserStatsResponse, *responses.InternalResponse) {
	var stats []responses.UserStatsResponse
	query := `
		SELECT 
		us.id,
		us.user_id,
		us.receiving_tasks_completed,
		us.picking_tasks_completed,
		us.avg_pick_time,
		us.pick_accuracy,
		us.total_picking_time,
		us.correct_picks,
		us.total_picks,
		u.first_name || ' ' || u.last_name AS username,
		u.email,
		us.created_at,
		us.updated_at
		FROM user_stats us
		JOIN users u ON us.user_id = u.id
	`

	if err := r.DB.Raw(query).Scan(&stats).Error; err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener todas las estadísticas de usuario",
			Handled: false,
		}
	}

	return stats, nil
}
