package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// ReceivingTasksRepository defines persistence operations for receiving tasks.
type ReceivingTasksRepository interface {
	GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse)
	GetReceivingTaskByID(id string) (*database.ReceivingTask, *responses.InternalResponse)
	CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse
	UpdateReceivingTask(id string, data map[string]interface{}) *responses.InternalResponse
	ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse
	ExportReceivingTaskToExcel() ([]byte, *responses.InternalResponse)
	CompleteFullTask(id string, location, userId string) *responses.InternalResponse
	CompleteReceivingLine(id string, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse
	GenerateImportTemplate(language string) ([]byte, error)
	// LinkSupplier links or unlinks a supplier on a receiving task (S2 R2 E1.7).
	LinkSupplier(taskID string, supplierID *string) *responses.InternalResponse
}
