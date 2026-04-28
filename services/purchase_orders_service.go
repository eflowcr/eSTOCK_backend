package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// PurchaseOrdersService provides business logic for Purchase Orders (PO1 + PO2 + PO3).
type PurchaseOrdersService struct {
	Repository ports.PurchaseOrdersRepository
}

func NewPurchaseOrdersService(repo ports.PurchaseOrdersRepository) *PurchaseOrdersService {
	return &PurchaseOrdersService{Repository: repo}
}

// Create creates a new draft purchase order scoped to tenantID.
// Server-side stamps: tenant_id from config, created_by from JWT user_id.
func (s *PurchaseOrdersService) Create(tenantID, createdBy string, req *requests.CreatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return s.Repository.Create(tenantID, createdBy, req)
}

// GetByID returns a purchase order by ID, scoped to tenantID.
func (s *PurchaseOrdersService) GetByID(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return s.Repository.GetByID(id, tenantID)
}

// List returns purchase orders for a tenant with optional filters and pagination.
func (s *PurchaseOrdersService) List(tenantID string, status, supplierID, search *string, from, to *string, limit, offset int) ([]responses.PurchaseOrderView, *responses.InternalResponse) {
	return s.Repository.List(tenantID, status, supplierID, search, from, to, limit, offset)
}

// Update patches mutable fields on a draft PO (enforces status=draft at repo layer).
func (s *PurchaseOrdersService) Update(id, tenantID string, req *requests.UpdatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return s.Repository.Update(id, tenantID, req)
}

// SoftDelete soft-deletes a purchase order scoped to tenantID.
func (s *PurchaseOrdersService) SoftDelete(id, tenantID string) *responses.InternalResponse {
	return s.Repository.SoftDelete(id, tenantID)
}

// Submit transitions a draft PO to 'submitted' and auto-generates a receiving task.
// Returns the updated PO view and the new receiving_task_id.
func (s *PurchaseOrdersService) Submit(id, tenantID, userID string) (*responses.PurchaseOrderView, string, *responses.InternalResponse) {
	return s.Repository.Submit(id, tenantID, userID)
}

// Cancel transitions a non-completed PO to 'cancelled'.
func (s *PurchaseOrdersService) Cancel(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return s.Repository.Cancel(id, tenantID)
}
