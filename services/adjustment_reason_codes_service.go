package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type AdjustmentReasonCodesService struct {
	Repository ports.AdjustmentReasonCodesRepository
}

func NewAdjustmentReasonCodesService(repo ports.AdjustmentReasonCodesRepository) *AdjustmentReasonCodesService {
	return &AdjustmentReasonCodesService{Repository: repo}
}

func (s *AdjustmentReasonCodesService) ListAdjustmentReasonCodes() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return s.Repository.ListAdjustmentReasonCodes()
}

func (s *AdjustmentReasonCodesService) ListAdjustmentReasonCodesAdmin() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return s.Repository.ListAdjustmentReasonCodesAdmin()
}

func (s *AdjustmentReasonCodesService) GetAdjustmentReasonCodeByID(id string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return s.Repository.GetAdjustmentReasonCodeByID(id)
}

func (s *AdjustmentReasonCodesService) GetAdjustmentReasonCodeByCode(code string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return s.Repository.GetAdjustmentReasonCodeByCode(code)
}

func (s *AdjustmentReasonCodesService) CreateAdjustmentReasonCode(req *requests.AdjustmentReasonCodeCreate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return s.Repository.CreateAdjustmentReasonCode(req)
}

func (s *AdjustmentReasonCodesService) UpdateAdjustmentReasonCode(id string, req *requests.AdjustmentReasonCodeUpdate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	return s.Repository.UpdateAdjustmentReasonCode(id, req)
}

func (s *AdjustmentReasonCodesService) DeleteAdjustmentReasonCode(id string) *responses.InternalResponse {
	return s.Repository.DeleteAdjustmentReasonCode(id)
}
