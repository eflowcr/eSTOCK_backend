package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// ClientsRepository defines persistence operations for clients (suppliers/customers).
type ClientsRepository interface {
	Create(tenantID string, data *requests.CreateClientRequest, createdBy *string) (*database.Client, *responses.InternalResponse)
	// GetByID performs a cross-tenant lookup for internal validation (e.g. picking/receiving task
	// customer/supplier checks). Use GetByIDForTenant for HTTP endpoint responses (HR1-M3).
	GetByID(id string) (*database.Client, *responses.InternalResponse)
	// GetByIDForTenant scopes the lookup to tenantID — prevents cross-tenant enumeration (HR1-M3).
	GetByIDForTenant(id, tenantID string) (*database.Client, *responses.InternalResponse)
	GetByTenantAndCode(tenantID, code string) (*database.Client, *responses.InternalResponse)
	ListByTenant(tenantID string) ([]database.Client, *responses.InternalResponse)
	// ListByTenantFiltered pushes optional type/isActive/search filters and pagination to SQL (M8).
	// Pass nil for any param to skip that filter.
	ListByTenantFiltered(tenantID string, clientType *string, isActive *bool, search *string, limit *int32, offset *int32) ([]database.Client, *responses.InternalResponse)
	// Update requires tenantID to prevent cross-tenant update (HR1-M3).
	Update(id string, data *requests.UpdateClientRequest, tenantID string) (*database.Client, *responses.InternalResponse)
	// SoftDelete requires tenantID to prevent cross-tenant soft-delete (HR1-M3).
	SoftDelete(id, tenantID string) *responses.InternalResponse
}
