package services

import (
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// InventoryCountsService orchestrates the count-sheet workflow:
//   draft → in_progress (Start) → in_review → submitted (creates adjustment) | cancelled.
//
// Service holds a reference to AdjustmentsRepository so Submit can persist the adjustment
// rolling up all variance lines. AdjustmentsRepo is optional — when nil, Submit returns the
// count without creating an adjustment (useful in tests).
type InventoryCountsService struct {
	Repository       ports.InventoryCountsRepository
	AdjustmentsRepo  ports.AdjustmentsRepository
}

func NewInventoryCountsService(repo ports.InventoryCountsRepository, adjRepo ports.AdjustmentsRepository) *InventoryCountsService {
	return &InventoryCountsService{Repository: repo, AdjustmentsRepo: adjRepo}
}

func (s *InventoryCountsService) List(status, locationID string) ([]database.InventoryCount, *responses.InternalResponse) {
	return s.Repository.List(status, locationID)
}

func (s *InventoryCountsService) GetDetail(id string) (*responses.InventoryCountDetail, *responses.InternalResponse) {
	return s.Repository.GetDetail(id)
}

func (s *InventoryCountsService) Create(userID string, req *requests.CreateInventoryCount) (*database.InventoryCount, *responses.InternalResponse) {
	return s.Repository.Create(userID, req)
}

// Start moves a count from draft|scheduled → in_progress.
// Caller-side authorization (creator or admin) is enforced by the controller.
func (s *InventoryCountsService) Start(id string) *responses.InternalResponse {
	c, resp := s.Repository.GetByID(id)
	if resp != nil {
		return resp
	}
	if c.Status != "draft" && c.Status != "scheduled" {
		return &responses.InternalResponse{
			Message:    "El conteo no puede iniciarse en su estado actual: " + c.Status,
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	return s.Repository.MarkStarted(id)
}

func (s *InventoryCountsService) Cancel(id string) *responses.InternalResponse {
	c, resp := s.Repository.GetByID(id)
	if resp != nil {
		return resp
	}
	if c.Status == "submitted" || c.Status == "cancelled" {
		return &responses.InternalResponse{
			Message:    "El conteo ya está cerrado y no puede cancelarse",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}
	return s.Repository.MarkCancelled(id)
}

// ScanLine resolves SKU (from sku or barcode), looks up expected qty, computes variance,
// persists the line, and returns it. The count must be in_progress.
func (s *InventoryCountsService) ScanLine(countID, userID string, req *requests.ScanCountLine) (*database.InventoryCountLine, *responses.InternalResponse) {
	c, resp := s.Repository.GetByID(countID)
	if resp != nil {
		return nil, resp
	}
	if c.Status != "in_progress" && c.Status != "in_review" {
		return nil, &responses.InternalResponse{
			Message:    "Solo se pueden registrar líneas en conteos en progreso",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	sku := strings.TrimSpace(req.SKU)
	if sku == "" {
		if strings.TrimSpace(req.Barcode) == "" {
			return nil, &responses.InternalResponse{
				Message:    "Se requiere SKU o código de barras",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		resolved, resolveResp := s.Repository.ResolveSKUByBarcode(req.Barcode)
		if resolveResp != nil {
			return nil, resolveResp
		}
		sku = resolved
	}

	locCode, resp := s.Repository.GetLocationCodeByID(req.LocationID)
	if resp != nil {
		return nil, resp
	}

	expected, resp := s.Repository.GetExpectedQty(sku, locCode, req.Lot)
	if resp != nil {
		return nil, resp
	}

	var lotPtr, serialPtr, notePtr *string
	if req.Lot != "" {
		v := req.Lot
		lotPtr = &v
	}
	if req.Serial != "" {
		v := req.Serial
		serialPtr = &v
	}
	if req.Note != "" {
		v := req.Note
		notePtr = &v
	}

	line := &database.InventoryCountLine{
		CountID:     countID,
		LocationID:  req.LocationID,
		SKU:         sku,
		Lot:         lotPtr,
		Serial:      serialPtr,
		ExpectedQty: expected,
		ScannedQty:  req.ScannedQty,
		VarianceQty: req.ScannedQty - expected,
		Note:        notePtr,
		ScannedBy:   userID,
	}
	if resp := s.Repository.AddLine(line); resp != nil {
		return nil, resp
	}
	return line, nil
}

// Submit aggregates variance lines into adjustments and closes the count.
//
// Each non-zero variance line yields a CreateAdjustment call (one adjustment per line —
// keeps things idempotent w.r.t. existing reason-code/direction logic in the
// AdjustmentsService). The first adjustment ID is recorded against the count for traceability.
//
// When AdjustmentsRepo is nil (tests), no adjustments are created but the count is still
// transitioned to "submitted".
func (s *InventoryCountsService) Submit(id, userID string) (*database.InventoryCount, *responses.InternalResponse) {
	c, resp := s.Repository.GetByID(id)
	if resp != nil {
		return nil, resp
	}
	if c.Status != "in_progress" && c.Status != "in_review" {
		return nil, &responses.InternalResponse{
			Message:    "Solo se pueden enviar conteos en progreso o en revisión",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	lines, resp := s.Repository.ListLines(id)
	if resp != nil {
		return nil, resp
	}

	firstAdjID := ""
	if s.AdjustmentsRepo != nil {
		for _, line := range lines {
			if line.VarianceQty == 0 {
				continue
			}
			locCode, locResp := s.Repository.GetLocationCodeByID(line.LocationID)
			if locResp != nil {
				return nil, locResp
			}

			direction := "inbound" // positive variance: stock found extra → add
			absQty := line.VarianceQty
			if absQty < 0 {
				direction = "outbound"
				absQty = -absQty
			}

			notes := "inventory_count " + c.Code
			adj := requests.CreateAdjustment{
				SKU:                line.SKU,
				Location:           locCode,
				AdjustmentQuantity: absQty,
				Reason:             "INVENTORY_COUNT_" + strings.ToUpper(direction),
				Notes:              notes,
			}
			created, adjResp := s.AdjustmentsRepo.CreateAdjustment(userID, adj)
			if adjResp != nil {
				return nil, adjResp
			}
			if firstAdjID == "" && created != nil {
				firstAdjID = created.ID
			}
		}
	}

	if resp := s.Repository.MarkSubmitted(id, userID, firstAdjID); resp != nil {
		return nil, resp
	}
	updated, _ := s.Repository.GetByID(id)
	return updated, nil
}
