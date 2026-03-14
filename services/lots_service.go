package services

import (
	"sort"
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type LotsService struct {
	Repository     ports.LotsRepository
	ArticlesRepo   ports.ArticlesRepository // optional: when set, GetLotsBySKU returns lots in rotation order (FIFO/FEFO)
}

// NewLotsService builds the lots service. articlesRepo may be nil; when set, GetLotsBySKU orders lots by article rotation strategy.
func NewLotsService(repo ports.LotsRepository, articlesRepo ports.ArticlesRepository) *LotsService {
	return &LotsService{
		Repository:   repo,
		ArticlesRepo: articlesRepo,
	}
}

func (s *LotsService) GetAllLots() ([]database.Lot, *responses.InternalResponse) {
	return s.Repository.GetAllLots()
}

func (s *LotsService) GetLotsBySKU(sku *string) ([]database.Lot, *responses.InternalResponse) {
	lots, resp := s.Repository.GetLotsBySKU(sku)
	if resp != nil || lots == nil || len(lots) == 0 {
		return lots, resp
	}
	if s.ArticlesRepo != nil && sku != nil && *sku != "" {
		article, errResp := s.ArticlesRepo.GetBySku(*sku)
		if errResp == nil && article != nil {
			rotationStrategy := strings.TrimSpace(strings.ToLower(article.RotationStrategy))
			if rotationStrategy != "fifo" && rotationStrategy != "fefo" {
				rotationStrategy = "fifo"
			}
			sortLotsByRotationStrategy(lots, rotationStrategy)
		}
	}
	return lots, nil
}

// sortLotsByRotationStrategy orders lots for picking/receiving: FIFO = oldest first, FEFO = earliest expiry first.
func sortLotsByRotationStrategy(lots []database.Lot, strategy string) {
	if strategy == "fefo" {
		sort.Slice(lots, func(i, j int) bool {
			ei, ej := lots[i].ExpirationDate, lots[j].ExpirationDate
			if ei == nil && ej == nil {
				return lots[i].CreatedAt.Before(lots[j].CreatedAt)
			}
			if ei == nil {
				return false // nulls last
			}
			if ej == nil {
				return true
			}
			if ei.Before(*ej) {
				return true
			}
			if ej.Before(*ei) {
				return false
			}
			return lots[i].CreatedAt.Before(lots[j].CreatedAt)
		})
		return
	}
	// FIFO: oldest first (created_at ascending)
	sort.Slice(lots, func(i, j int) bool {
		return lots[i].CreatedAt.Before(lots[j].CreatedAt)
	})
}

func (s *LotsService) Create(data *requests.CreateLotRequest) *responses.InternalResponse {
	return s.Repository.CreateLot(data)
}

func (s *LotsService) UpdateUpdateLot(id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateLot(id, data)
}

func (s *LotsService) DeleteLot(id string) *responses.InternalResponse {
	return s.Repository.DeleteLot(id)
}
