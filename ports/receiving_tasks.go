package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// ReceivingTasksRepository defines persistence operations for receiving tasks.
type ReceivingTasksRepository interface {
	// GetAllReceivingTasks returns all receiving tasks without tenant filter.
	// internal use only — bypass tenant. Prefer GetAllForTenant in HTTP handlers.
	GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse)
	// GetAllForTenant returns receiving tasks scoped to a specific tenant (S2.5 M3.1).
	GetAllForTenant(tenantID string) ([]responses.ReceivingTasksView, *responses.InternalResponse)
	GetReceivingTaskByID(id string) (*database.ReceivingTask, *responses.InternalResponse)
	CreateReceivingTask(userId string, tenantID string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse
	UpdateReceivingTask(id string, data map[string]interface{}) *responses.InternalResponse
	ImportReceivingTaskFromExcel(userID string, tenantID string, fileBytes []byte) *responses.InternalResponse
	ExportReceivingTaskToExcel(tenantID string) ([]byte, *responses.InternalResponse)
	CompleteFullTask(id string, location, userId string) *responses.InternalResponse
	CompleteReceivingLine(id string, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse
	GenerateImportTemplate(language string) ([]byte, error)
	// LinkSupplier links or unlinks a supplier on a receiving task (S2 R2 E1.7).
	LinkSupplier(taskID string, supplierID *string) *responses.InternalResponse
}
