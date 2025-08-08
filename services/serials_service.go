package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type SerialsService struct {
	Repository *repositories.SerialsRepository
}

func (s *SerialsService) GetSerialByID(id int) (*database.Serial, *responses.InternalResponse) {
	return s.Repository.GetSerialByID(id)
}

func NewSerialsService(repo *repositories.SerialsRepository) *SerialsService {
	return &SerialsService{Repository: repo}
}

func (s *SerialsService) GetSerialsBySKU(sku string) ([]database.Serial, *responses.InternalResponse) {
	return s.Repository.GetSerialsBySKU(sku)
}

func (s *SerialsService) Create(data *requests.CreateSerialRequest) *responses.InternalResponse {
	return s.Repository.CreateSerial(data)
}

func (s *SerialsService) UpdateSerial(id int, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateSerial(id, data)
}

func (s *SerialsService) Delete(id int) *responses.InternalResponse {
	return s.Repository.DeleteSerial(id)
}

