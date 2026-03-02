package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// PickingTaskRepository defines persistence operations for picking tasks.
type PickingTaskRepository interface {
	GetAllPickingTasks() ([]responses.PickingTaskView, *responses.InternalResponse)
	GetPickingTaskByID(id int) (*database.PickingTask, *responses.InternalResponse)
	CreatePickingTask(userId string, task *requests.CreatePickingTaskRequest) *responses.InternalResponse
	UpdatePickingTask(id int, data map[string]interface{}) *responses.InternalResponse
	ImportPickingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse
	ExportPickingTasksToExcel() ([]byte, *responses.InternalResponse)
	CompletePickingTask(id int, location, userId string) *responses.InternalResponse
	CompletePickingLine(id int, location, userId string, item requests.PickingTaskItemRequest) *responses.InternalResponse
}
