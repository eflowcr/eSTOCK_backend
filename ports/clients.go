package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// ClientsRepository defines persistence operations for clients (suppliers/customers).
type ClientsRepository interface {
	Create(tenantID string, data *requests.CreateClientRequest, createdBy *string) (*database.Client, *responses.InternalResponse)
	GetByID(id string) (*database.Client, *responses.InternalResponse)
	GetByTenantAndCode(tenantID, code string) (*database.Client, *responses.InternalResponse)
	ListByTenant(tenantID string) ([]database.Client, *responses.InternalResponse)
	Update(id string, data *requests.UpdateClientRequest) (*database.Client, *responses.InternalResponse)
	SoftDelete(id string) *responses.InternalResponse
}
