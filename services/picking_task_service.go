package services

import (
	"context"
	"fmt"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type PickingTaskService struct {
	Repository     ports.PickingTaskRepository
	ClientsService clientLookup // optional: validate customer_id on create/link (S2 R2)
}

func NewPickingTaskService(repo ports.PickingTaskRepository) *PickingTaskService {
	return &PickingTaskService{Repository: repo}
}

// WithClientsService attaches an optional ClientsService for customer validation.
func (s *PickingTaskService) WithClientsService(cs clientLookup) *PickingTaskService {
	s.ClientsService = cs
	return s
}

// GetAllPickingTasks returns all picking tasks (no tenant filter).
// internal use only — bypass tenant. Prefer ListByTenant in HTTP handlers.
func (s *PickingTaskService) GetAllPickingTasks() ([]responses.PickingTaskView, *responses.InternalResponse) {
	return s.Repository.GetAllPickingTasks()
}

// ListByTenant returns picking tasks scoped to a specific tenant (S2.5 M3.1).
func (s *PickingTaskService) ListByTenant(tenantID string) ([]responses.PickingTaskView, *responses.InternalResponse) {
	return s.Repository.GetAllForTenant(tenantID)
}

func (s *PickingTaskService) GetPickingTaskByID(id string) (*database.PickingTask, *responses.InternalResponse) {
	return s.Repository.GetPickingTaskByID(id)
}

func (s *PickingTaskService) CreatePickingTask(userId string, tenantID string, task *requests.CreatePickingTaskRequest) *responses.InternalResponse {
	if task.CustomerID != nil && *task.CustomerID != "" {
		if resp := s.validateCustomer(*task.CustomerID); resp != nil {
			return resp
		}
	}
	return s.Repository.CreatePickingTask(userId, tenantID, task)
}

func (s *PickingTaskService) StartPickingTask(ctx context.Context, id, userId string) *responses.InternalResponse {
	return s.Repository.StartPickingTask(ctx, id, userId)
}

func (s *PickingTaskService) UpdatePickingTask(ctx context.Context, id string, data map[string]interface{}, userId string) *responses.InternalResponse {
	return s.Repository.UpdatePickingTask(ctx, id, data, userId)
}

func (s *PickingTaskService) ImportPickingTaskFromExcel(userID string, tenantID string, fileBytes []byte) *responses.InternalResponse {
	return s.Repository.ImportPickingTaskFromExcel(userID, tenantID, fileBytes)
}

func (s *PickingTaskService) ExportPickingTasksToExcel(tenantID string) ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportPickingTasksToExcel(tenantID)
}

func (s *PickingTaskService) CompletePickingTask(ctx context.Context, id, userId string) *responses.InternalResponse {
	return s.Repository.CompletePickingTask(ctx, id, userId)
}

func (s *PickingTaskService) CompletePickingLine(ctx context.Context, id, userId string, item requests.PickingTaskItemRequest) *responses.InternalResponse {
	return s.Repository.CompletePickingLine(ctx, id, userId, item)
}

func (s *PickingTaskService) GenerateImportTemplate(language string) ([]byte, error) {
	return s.Repository.GenerateImportTemplate(language)
}

// LinkCustomer links or unlinks a customer on a picking task.
func (s *PickingTaskService) LinkCustomer(taskID string, customerID *string) *responses.InternalResponse {
	if customerID != nil && *customerID != "" {
		if resp := s.validateCustomer(*customerID); resp != nil {
			return resp
		}
	}
	return s.Repository.LinkCustomer(taskID, customerID)
}

// validateCustomer checks that the client exists and is type customer or both.
func (s *PickingTaskService) validateCustomer(customerID string) *responses.InternalResponse {
	if s.ClientsService == nil {
		return nil // ClientsService not wired — skip validation
	}
	client, resp := s.ClientsService.GetByID(customerID)
	if resp != nil {
		return &responses.InternalResponse{
			Message:    fmt.Sprintf("customer_id inválido: %s", resp.Message),
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	if client.Type != "customer" && client.Type != "both" {
		return &responses.InternalResponse{
			Message:    fmt.Sprintf("el cliente '%s' es de tipo '%s', se requiere 'customer' o 'both'", customerID, client.Type),
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	return nil
}
