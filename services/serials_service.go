package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// SerialsService is a thin pass-through to the tenant-aware repository.
// S3.5 W2-A: every method now requires tenantID.
type SerialsService struct {
	Repository ports.SerialsRepository
}

func NewSerialsService(repo ports.SerialsRepository) *SerialsService {
	return &SerialsService{Repository: repo}
}

func (s *SerialsService) GetSerialByID(tenantID, id string) (*database.Serial, *responses.InternalResponse) {
	return s.Repository.GetSerialByID(tenantID, id)
}

func (s *SerialsService) GetSerialsBySKU(tenantID, sku string) ([]database.Serial, *responses.InternalResponse) {
	return s.Repository.GetSerialsBySKU(tenantID, sku)
}

func (s *SerialsService) Create(tenantID string, data *requests.CreateSerialRequest) *responses.InternalResponse {
	return s.Repository.CreateSerial(tenantID, data)
}

func (s *SerialsService) UpdateSerial(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateSerial(tenantID, id, data)
}

func (s *SerialsService) Delete(tenantID, id string) *responses.InternalResponse {
	return s.Repository.DeleteSerial(tenantID, id)
}
