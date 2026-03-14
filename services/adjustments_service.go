package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type AdjustmentsService struct {
	Repository         ports.AdjustmentsRepository
	ReasonCodesRepository ports.AdjustmentReasonCodesRepository
}

func NewAdjustmentsService(repo ports.AdjustmentsRepository, reasonCodesRepo ports.AdjustmentReasonCodesRepository) *AdjustmentsService {
	return &AdjustmentsService{
		Repository:         repo,
		ReasonCodesRepository: reasonCodesRepo,
	}
}

func (s *AdjustmentsService) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	return s.Repository.GetAllAdjustments()
}

func (s *AdjustmentsService) GetAdjustmentByID(id string) (*database.Adjustment, *responses.InternalResponse) {
	return s.Repository.GetAdjustmentByID(id)
}

func (s *AdjustmentsService) GetAdjustmentDetails(id string) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	return s.Repository.GetAdjustmentDetails(id)
}

func (s *AdjustmentsService) CreateAdjustment(userId string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse) {
	// Quantity must be non-negative; sign is determined by reason code direction.
	if adjustment.AdjustmentQuantity < 0 {
		return nil, &responses.InternalResponse{
			Message:    "adjustment quantity must be zero or positive; add or subtract is determined by the reason code",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	signedQuantity := adjustment.AdjustmentQuantity
	if s.ReasonCodesRepository != nil {
		reasonCode, resp := s.ReasonCodesRepository.GetAdjustmentReasonCodeByCode(adjustment.Reason)
		if resp != nil {
			return nil, resp
		}
		if reasonCode == nil {
			return nil, &responses.InternalResponse{
				Message:    "invalid or inactive reason code for adjustment",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		if reasonCode.Direction == "outbound" {
			signedQuantity = -adjustment.AdjustmentQuantity
		}
		// inbound: keep positive
	}
	req := adjustment
	req.AdjustmentQuantity = signedQuantity
	return s.Repository.CreateAdjustment(userId, req)
}

func (s *AdjustmentsService) ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportAdjustmentsToExcel()
}
