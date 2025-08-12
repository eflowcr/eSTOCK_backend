package repositories

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"gorm.io/gorm"
)

type ReceivingTasksRepository struct {
	DB *gorm.DB
}

func (r *ReceivingTasksRepository) GetAllReceivingTasks() ([]database.ReceivingTask, *responses.InternalResponse) {
	var tasks []database.ReceivingTask

	err := r.DB.
		Table(database.ReceivingTask{}.TableName()).
		Order("created_at DESC").
		Find(&tasks).Error

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Failed to fetch receiving tasks",
			Handled: false,
		}
	}

	if len(tasks) == 0 {
		return nil, &responses.InternalResponse{
			Error:   nil,
			Message: "No receiving tasks found",
			Handled: true,
		}
	}

	return tasks, nil
}
