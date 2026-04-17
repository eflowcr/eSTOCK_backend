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

func (s *ClientsService) GetByID(id string) (*database.Client, *responses.InternalResponse) {
	return s.Repository.GetByID(id)
}

func (s *ClientsService) List(tenantID string, clientType *string, isActive *bool, search *string) ([]database.Client, *responses.InternalResponse) {
	all, resp := s.Repository.ListByTenant(tenantID)
	if resp != nil {
		return nil, resp
	}

	var filtered []database.Client
	for _, c := range all {
		if clientType != nil && c.Type != *clientType {
			continue
		}
		if isActive != nil && c.IsActive != *isActive {
			continue
		}
		if search != nil && *search != "" {
			q := *search
			if !containsIgnoreCase(c.Name, q) && !containsIgnoreCase(c.Code, q) {
				continue
			}
		}
		filtered = append(filtered, c)
	}
	if filtered == nil {
		filtered = []database.Client{}
	}
	return filtered, nil
}

func (s *ClientsService) Update(id string, data *requests.UpdateClientRequest, tenantID string) (*database.Client, *responses.InternalResponse) {
	existing, resp := s.Repository.GetByID(id)
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

	return s.Repository.Update(id, data)
}

func (s *ClientsService) SoftDelete(id string) *responses.InternalResponse {
	return s.Repository.SoftDelete(id)
}
