package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// BackordersService provides business logic for backorders (BO1 + BO2).
type BackordersService struct {
	Repository ports.BackordersRepository
}

func NewBackordersService(repo ports.BackordersRepository) *BackordersService {
	return &BackordersService{Repository: repo}
}

// List returns paginated backorders for a tenant with optional status/SO filters.
func (s *BackordersService) List(tenantID string, status, soID *string, page, limit int) (*responses.BackorderListResponse, *responses.InternalResponse) {
	return s.Repository.List(tenantID, status, soID, page, limit)
}

// GetByID returns a single backorder by ID, scoped to tenantID.
func (s *BackordersService) GetByID(id, tenantID string) (*responses.BackorderResponse, *responses.InternalResponse) {
	return s.Repository.GetByID(id, tenantID)
}

// Fulfill creates a new picking task for a pending backorder (BO2).
// Enforces max depth=1 — backorder-sourced picking tasks will not generate further backorders.
func (s *BackordersService) Fulfill(id, tenantID, userID string) (*responses.FulfillBackorderResult, *responses.InternalResponse) {
	return s.Repository.Fulfill(id, tenantID, userID)
}
