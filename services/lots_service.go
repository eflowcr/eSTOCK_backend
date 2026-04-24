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
	Repository   ports.LotsRepository
	ArticlesRepo ports.ArticlesRepository // optional: when set, GetLotsBySKU returns lots in rotation order (FIFO/FEFO)
}

// NewLotsService builds the lots service. articlesRepo may be nil; when set, GetLotsBySKU orders lots by article rotation strategy.
func NewLotsService(repo ports.LotsRepository, articlesRepo ports.ArticlesRepository) *LotsService {
	return &LotsService{
		Repository:   repo,
		ArticlesRepo: articlesRepo,
	}
}

// S3.5 W2-B: every public method takes tenantID so the controller can pass Config.TenantID
// (or middleware-resolved tenant context) and isolation is enforced one layer below the HTTP boundary.

func (s *LotsService) GetAllLots(tenantID string) ([]database.Lot, *responses.InternalResponse) {
	return s.Repository.GetAllLots(tenantID)
}

func (s *LotsService) GetLotByID(tenantID, id string) (*database.Lot, *responses.InternalResponse) {
	return s.Repository.GetLotByIDForTenant(id, tenantID)
}

// GetTrace returns the full provenance trace for a lot owned by tenantID.
func (s *LotsService) GetTrace(tenantID, lotID string) (*responses.LotTraceResponse, *responses.InternalResponse) {
	return s.Repository.GetLotTrace(tenantID, lotID)
}

func (s *LotsService) GetLotsBySKU(tenantID string, sku *string) ([]database.Lot, *responses.InternalResponse) {
	lots, resp := s.Repository.GetLotsBySKU(tenantID, sku)
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

func (s *LotsService) Create(tenantID string, data *requests.CreateLotRequest) *responses.InternalResponse {
	return s.Repository.CreateLot(tenantID, data)
}

func (s *LotsService) UpdateUpdateLot(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	return s.Repository.UpdateLot(tenantID, id, data)
}

func (s *LotsService) DeleteLot(tenantID, id string) *responses.InternalResponse {
	return s.Repository.DeleteLot(tenantID, id)
}
