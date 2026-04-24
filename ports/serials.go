package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// SerialsRepository defines persistence operations for serials.
//
// S3.5 W2-A: all methods are tenant-scoped. Cross-tenant lookups are not
// exposed here; internal article-update warning logic uses the global
// ListSerialsBySku query directly via the sqlc client.
type SerialsRepository interface {
	GetSerialByID(tenantID, id string) (*database.Serial, *responses.InternalResponse)
	GetSerialsBySKU(tenantID, sku string) ([]database.Serial, *responses.InternalResponse)
	CreateSerial(tenantID string, data *requests.CreateSerialRequest) *responses.InternalResponse
	UpdateSerial(tenantID, id string, data map[string]interface{}) *responses.InternalResponse
	DeleteSerial(tenantID, id string) *responses.InternalResponse
}
