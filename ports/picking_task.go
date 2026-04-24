package ports

import (
	"context"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// PickingTaskRepository defines persistence operations for picking tasks.
type PickingTaskRepository interface {
	// GetAllPickingTasks returns all picking tasks without tenant filter.
	// internal use only — bypass tenant. Prefer GetAllForTenant in HTTP handlers.
	GetAllPickingTasks() ([]responses.PickingTaskView, *responses.InternalResponse)
	// GetAllForTenant returns picking tasks scoped to a specific tenant (S2.5 M3.1).
	GetAllForTenant(tenantID string) ([]responses.PickingTaskView, *responses.InternalResponse)
	GetPickingTaskByID(id string) (*database.PickingTask, *responses.InternalResponse)
	CreatePickingTask(userId string, tenantID string, task *requests.CreatePickingTaskRequest) *responses.InternalResponse
	// StartPickingTask transitions the task to in_progress and applies lazy reservations (B3a).
	StartPickingTask(ctx context.Context, id, userId string) *responses.InternalResponse
	// UpdatePickingTask applies whitelist-filtered updates; recalculates reservations when items
	// change while task is in_progress, and releases them on cancel (B3b/B3c).
	UpdatePickingTask(ctx context.Context, id string, data map[string]interface{}, userId string) *responses.InternalResponse
	ImportPickingTaskFromExcel(userID string, tenantID string, fileBytes []byte) *responses.InternalResponse
	ExportPickingTasksToExcel() ([]byte, *responses.InternalResponse)
	// CompletePickingTask finalises all items using allocations (H5).
	// The old `location` parameter is removed; locations come from each item's allocations.
	CompletePickingTask(ctx context.Context, id, userId string) *responses.InternalResponse
	// CompletePickingLine finalises a single item using its allocations (B3d).
	CompletePickingLine(ctx context.Context, id, userId string, item requests.PickingTaskItemRequest) *responses.InternalResponse
	GenerateImportTemplate(language string) ([]byte, error)
	// LinkCustomer links or unlinks a customer on a picking task (S2 R2 E1.7).
	LinkCustomer(taskID string, customerID *string) *responses.InternalResponse
}
