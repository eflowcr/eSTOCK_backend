package ports

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
)

// AdjustmentsRepository defines persistence operations for adjustments.
type AdjustmentsRepository interface {
	GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse)
	GetAdjustmentByID(id int) (*database.Adjustment, *responses.InternalResponse)
	GetAdjustmentDetails(id int) (*dto.AdjustmentDetails, *responses.InternalResponse)
	CreateAdjustment(userId string, adjustment requests.CreateAdjustment) *responses.InternalResponse
	ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse)
}
