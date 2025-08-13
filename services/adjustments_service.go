package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type AdjustmentsService struct {
	Repository *repositories.AdjustmentsRepository
}

func NewAdjustmentsService(repo *repositories.AdjustmentsRepository) *AdjustmentsService {
	return &AdjustmentsService{
		Repository: repo,
	}
}

func (s *AdjustmentsService) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	return s.Repository.GetAllAdjustments()
}

func (s *AdjustmentsService) GetAdjustmentByID(id int) (*database.Adjustment, *responses.InternalResponse) {
	return s.Repository.GetAdjustmentByID(id)
}

func (s *AdjustmentsService) GetAdjustmentDetails(id int) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	return s.Repository.GetAdjustmentDetails(id)
}
