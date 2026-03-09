package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type AdjustmentsService struct {
	Repository ports.AdjustmentsRepository
}

func NewAdjustmentsService(repo ports.AdjustmentsRepository) *AdjustmentsService {
	return &AdjustmentsService{
		Repository: repo,
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

func (s *AdjustmentsService) CreateAdjustment(userId string, adjustment requests.CreateAdjustment) *responses.InternalResponse {
	return s.Repository.CreateAdjustment(userId, adjustment)
}

func (s *AdjustmentsService) ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportAdjustmentsToExcel()
}
