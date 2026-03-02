package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// ReceivingTasksRepository defines persistence operations for receiving tasks.
type ReceivingTasksRepository interface {
	GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse)
	GetReceivingTaskByID(id int) (*database.ReceivingTask, *responses.InternalResponse)
	CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse
	UpdateReceivingTask(id int, data map[string]interface{}) *responses.InternalResponse
	ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse
	ExportReceivingTaskToExcel() ([]byte, *responses.InternalResponse)
	CompleteFullTask(id int, location, userId string) *responses.InternalResponse
	CompleteReceivingLine(id int, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse
}
