package services

import (
	"fmt"
	"log"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// clientLookup is a narrow interface for client retrieval used for supplier/customer validation.
// Implemented by *ClientsService; also mockable in tests.
type clientLookup interface {
	GetByID(id string) (*database.Client, *responses.InternalResponse)
}

type ReceivingTasksService struct {
	Repository     ports.ReceivingTasksRepository
	ClientsService clientLookup // optional: validate supplier_id on create/link (S2 R2)
}

func NewReceivingTasksService(repo ports.ReceivingTasksRepository) *ReceivingTasksService {
	return &ReceivingTasksService{
		Repository: repo,
	}
}

// WithClientsService attaches an optional ClientsService for supplier validation.
func (s *ReceivingTasksService) WithClientsService(cs clientLookup) *ReceivingTasksService {
	s.ClientsService = cs
	return s
}

func (s *ReceivingTasksService) GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse) {
	return s.Repository.GetAllReceivingTasks()
}

func (s *ReceivingTasksService) GetReceivingTaskByID(id string) (*database.ReceivingTask, *responses.InternalResponse) {
	return s.Repository.GetReceivingTaskByID(id)
}

func (s *ReceivingTasksService) CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse {
	if task.SupplierID != nil && *task.SupplierID != "" {
		if resp := s.validateSupplier(*task.SupplierID); resp != nil {
			return resp
		}
	}
	return s.Repository.CreateReceivingTask(userId, task)
}

func (s *ReceivingTasksService) UpdateReceivingTask(id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateReceivingTask(id, data)
}

func (s *ReceivingTasksService) ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	return s.Repository.ImportReceivingTaskFromExcel(userID, fileBytes)
}

func (s *ReceivingTasksService) ExportReceivingTaskToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportReceivingTaskToExcel()
}

func (s *ReceivingTasksService) CompleteFullTask(id string, location, userId string) *responses.InternalResponse {
	return s.Repository.CompleteFullTask(id, location, userId)
}

// CompleteReceivingLine applies R1 backfill logic before delegating to the repository.
// If accepted_qty and rejected_qty are both nil/0 but received_qty > 0, accepted_qty is backfilled
// from received_qty to preserve backward compatibility with legacy callers.
func (s *ReceivingTasksService) CompleteReceivingLine(id string, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse {
	item = applyAcceptedRejectedBackfill(item)
	return s.Repository.CompleteReceivingLine(id, location, userId, item)
}

func (s *ReceivingTasksService) GenerateImportTemplate(language string) ([]byte, error) {
	return s.Repository.GenerateImportTemplate(language)
}

// LinkSupplier links or unlinks a supplier on a receiving task.
func (s *ReceivingTasksService) LinkSupplier(taskID string, supplierID *string) *responses.InternalResponse {
	if supplierID != nil && *supplierID != "" {
		if resp := s.validateSupplier(*supplierID); resp != nil {
			return resp
		}
	}
	return s.Repository.LinkSupplier(taskID, supplierID)
}

// validateSupplier checks that the client exists and is type supplier or both.
func (s *ReceivingTasksService) validateSupplier(supplierID string) *responses.InternalResponse {
	if s.ClientsService == nil {
		return nil // ClientsService not wired — skip validation (allows integration without pool)
	}
	client, resp := s.ClientsService.GetByID(supplierID)
	if resp != nil {
		return &responses.InternalResponse{
			Message:    fmt.Sprintf("supplier_id inválido: %s", resp.Message),
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	if client.Type != "supplier" && client.Type != "both" {
		return &responses.InternalResponse{
			Message:    fmt.Sprintf("el cliente '%s' es de tipo '%s', se requiere 'supplier' o 'both'", supplierID, client.Type),
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	return nil
}

// applyAcceptedRejectedBackfill implements R1 backward-compat logic:
// if accepted_qty and rejected_qty are both nil/0 but received_qty > 0 → accepted_qty = received_qty.
func applyAcceptedRejectedBackfill(item requests.ReceivingTaskItemRequest) requests.ReceivingTaskItemRequest {
	acceptedZero := item.AcceptedQty == nil || *item.AcceptedQty == 0
	rejectedZero := item.RejectedQty == nil || *item.RejectedQty == 0
	if acceptedZero && rejectedZero && item.ReceivedQuantity != nil && *item.ReceivedQuantity > 0 {
		v := float64(*item.ReceivedQuantity)
		item.AcceptedQty = &v
		log.Printf("[receiving] backfill accepted_qty=%.2f for SKU %s (legacy received_qty)", v, item.SKU)
	}
	return item
}
