package responses

import "time"

type UserStatsResponse struct {
	ID                      int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID                  string    `gorm:"column:user_id;unique" json:"user_id"`
	ReceivingTasksCompleted int       `gorm:"column:receiving_tasks_completed" json:"receiving_tasks_completed"`
	PickingTasksCompleted   int       `gorm:"column:picking_tasks_completed" json:"picking_tasks_completed"`
	AvgPickTime             int       `gorm:"column:avg_pick_time" json:"avg_pick_time"`
	PickAccuracy            int       `gorm:"column:pick_accuracy" json:"pick_accuracy"`
	TotalPickingTime        int       `gorm:"column:total_picking_time" json:"total_picking_time"`
	CorrectPicks            int       `gorm:"column:correct_picks" json:"correct_picks"`
	TotalPicks              int       `gorm:"column:total_picks" json:"total_picks"`
	Username                string    `gorm:"column:username" json:"username"`
	Email                   string    `gorm:"column:email" json:"email"`
	CreatedAt               time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}
