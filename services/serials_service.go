package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type SerialsService struct {
	Repository ports.SerialsRepository
}

func (s *SerialsService) GetSerialByID(id string) (*database.Serial, *responses.InternalResponse) {
	return s.Repository.GetSerialByID(id)
}

func NewSerialsService(repo ports.SerialsRepository) *SerialsService {
	return &SerialsService{Repository: repo}
}

func (s *SerialsService) GetSerialsBySKU(sku string) ([]database.Serial, *responses.InternalResponse) {
	return s.Repository.GetSerialsBySKU(sku)
}

func (s *SerialsService) Create(data *requests.CreateSerialRequest) *responses.InternalResponse {
	return s.Repository.CreateSerial(data)
}

func (s *SerialsService) UpdateSerial(id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateSerial(id, data)
}

func (s *SerialsService) Delete(id string) *responses.InternalResponse {
	return s.Repository.DeleteSerial(id)
}

