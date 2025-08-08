package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/repositories"
)

type LotsService struct {
	Repository *repositories.LotsRepository
}

func NewLotsService(repo *repositories.LotsRepository) *LotsService {
	return &LotsService{
		Repository: repo,
	}
}

func (s *LotsService) GetAllLots() ([]database.Lot, *responses.InternalResponse) {
	return s.Repository.GetAllLots()
}

func (s *LotsService) GetLotsBySKU(sku *string) ([]database.Lot, *responses.InternalResponse) {
	return s.Repository.GetLotsBySKU(sku)
}

func (s *LotsService) Create(data *requests.CreateLotRequest) *responses.InternalResponse {
	return s.Repository.CreateLot(data)
}

func (s *LotsService) UpdateUpdateLot(id int, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateLot(id, data)
}

func (s *LotsService) DeleteLot(id int) *responses.InternalResponse {
	return s.Repository.DeleteLot(id)
}
