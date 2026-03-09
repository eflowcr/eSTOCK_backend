package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// LotsRepository defines persistence operations for lots.
type LotsRepository interface {
	GetAllLots() ([]database.Lot, *responses.InternalResponse)
	GetLotsBySKU(sku *string) ([]database.Lot, *responses.InternalResponse)
	CreateLot(data *requests.CreateLotRequest) *responses.InternalResponse
	UpdateLot(id string, data map[string]interface{}) *responses.InternalResponse
	DeleteLot(id string) *responses.InternalResponse
}
