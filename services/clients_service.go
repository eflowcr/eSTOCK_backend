package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type ClientsService struct {
	Repository ports.ClientsRepository
}

func NewClientsService(repo ports.ClientsRepository) *ClientsService {
	return &ClientsService{Repository: repo}
}

func (s *ClientsService) Create(tenantID string, data *requests.CreateClientRequest, createdBy *string) (*database.Client, *responses.InternalResponse) {
	existing, resp := s.Repository.GetByTenantAndCode(tenantID, data.Code)
	if resp != nil {
		return nil, resp
	}
	if existing != nil {
		return nil, &responses.InternalResponse{
			Message:    "Ya existe un cliente con ese código",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		}
	}
	return s.Repository.Create(tenantID, data, createdBy)
}

// GetByID performs a lookup without tenant filter — satisfies the internal clientLookup
// interface used by picking/receiving task validation. For HTTP responses use GetByIDForTenant.
func (s *ClientsService) GetByID(id string) (*database.Client, *responses.InternalResponse) {
	return s.Repository.GetByID(id)
}

// GetByIDForTenant scopes the lookup to tenantID — use for HTTP endpoint responses (HR1-M3).
func (s *ClientsService) GetByIDForTenant(id, tenantID string) (*database.Client, *responses.InternalResponse) {
	return s.Repository.GetByIDForTenant(id, tenantID)
}

// List delegates type/isActive/search filtering and pagination to SQL (M8 — HR1 deferred).
// No in-memory filtering needed: the repository pushes all conditions to the DB.
func (s *ClientsService) List(tenantID string, clientType *string, isActive *bool, search *string) ([]database.Client, *responses.InternalResponse) {
	clients, resp := s.Repository.ListByTenantFiltered(tenantID, clientType, isActive, search, nil, nil)
	if resp != nil {
		return nil, resp
	}
	if clients == nil {
		clients = []database.Client{}
	}
	return clients, nil
}

func (s *ClientsService) Update(id string, data *requests.UpdateClientRequest, tenantID string) (*database.Client, *responses.InternalResponse) {
	// Existence check is tenant-scoped (HR1-M3).
	existing, resp := s.Repository.GetByIDForTenant(id, tenantID)
	if resp != nil {
		return nil, resp
	}

	// Check code uniqueness (if code changed)
	if existing.Code != data.Code {
		byCode, codeResp := s.Repository.GetByTenantAndCode(tenantID, data.Code)
		if codeResp != nil {
			return nil, codeResp
		}
		if byCode != nil && byCode.ID != id {
			return nil, &responses.InternalResponse{
				Message:    "Ya existe un cliente con ese código",
				Handled:    true,
				StatusCode: responses.StatusConflict,
			}
		}
	}

	return s.Repository.Update(id, data, tenantID)
}

// SoftDelete requires tenantID to prevent cross-tenant soft-delete (HR1-M3).
func (s *ClientsService) SoftDelete(id, tenantID string) *responses.InternalResponse {
	return s.Repository.SoftDelete(id, tenantID)
}
