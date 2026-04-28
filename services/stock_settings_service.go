package services

import (
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type StockSettingsService struct {
	Repository ports.StockSettingsRepository
}

func NewStockSettingsService(repo ports.StockSettingsRepository) *StockSettingsService {
	return &StockSettingsService{Repository: repo}
}

func (s *StockSettingsService) GetOrCreate(tenantID string) (*database.StockSetting, *responses.InternalResponse) {
	return s.Repository.GetOrCreate(tenantID)
}

func (s *StockSettingsService) Update(tenantID string, data *requests.UpdateStockSettingsRequest) (*database.StockSetting, *responses.InternalResponse) {
	return s.Repository.Upsert(tenantID, data)
}
