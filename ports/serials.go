package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// SerialsRepository defines persistence operations for serials.
type SerialsRepository interface {
	GetSerialByID(id int) (*database.Serial, *responses.InternalResponse)
	GetSerialsBySKU(sku string) ([]database.Serial, *responses.InternalResponse)
	CreateSerial(data *requests.CreateSerialRequest) *responses.InternalResponse
	UpdateSerial(id int, data map[string]interface{}) *responses.InternalResponse
	DeleteSerial(id int) *responses.InternalResponse
}
