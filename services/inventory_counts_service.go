package services

import (
	"strings"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

// InventoryCountsService orchestrates the count-sheet workflow:
//   draft → in_progress (Start) → in_review → submitted (creates adjustments) | cancelled.
//
// AdjustmentsCreator is the narrow port that flips signs based on reason code and
// persists the actual stock mutation. When nil, Submit transitions the count to
// "submitted" without creating any adjustments (test-only path; in production the
// wire layer must always inject a real creator — see wire.NewInventoryCounts).
//
// Variance handling: variance is computed at scan-time AND recomputed at submit-time
// (inside the repo's SubmitWithAdjustments transaction). This is the "recompute at
// submit" approach — see N2-2 in the W0 hostile review. Concurrent stock movement
// between scan and submit is tolerated because the persisted variance always
// reflects the latest expected_qty at the moment of submit.
type InventoryCountsService struct {
	Repository         ports.InventoryCountsRepository
	AdjustmentsCreator ports.InventoryAdjustmentsCreator
}

func NewInventoryCountsService(repo ports.InventoryCountsRepository, creator ports.InventoryAdjustmentsCreator) *InventoryCountsService {
	return &InventoryCountsService{Repository: repo, AdjustmentsCreator: creator}
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

// Cancel transitions a non-closed count to "cancelled". Only the creator or an
// admin may cancel a count (W0 hostile review N2-1: prevent operator griefing).
func (s *InventoryCountsService) Cancel(id, callerUserID, callerRole string) *responses.InternalResponse {
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
	if !isCountOwnerOrAdmin(c, callerUserID, callerRole) {
		return forbidCountAction("cancelar")
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
// The orchestration (re-fetch expected qty per line → recompute variance →
// fan out one CreateAdjustment per non-zero line → MarkSubmitted) runs inside
// a single GORM transaction at the repository layer (W0 hostile review N1-2:
// atomicity — partial failure rolls back every adjustment plus the state
// transition). The repo delegates per-line adjustment creation to
// AdjustmentsCreator.CreateAdjustmentTx which is responsible for the reason-code
// driven sign flip (W0 hostile review N1-1).
//
// Ownership: only the count creator or an admin may submit (W0 hostile review N2-1).
//
// When AdjustmentsCreator is nil (tests only), the count is transitioned to
// submitted without any stock mutation — the production wire layer always
// injects a real creator.
func (s *InventoryCountsService) Submit(id, userID, callerRole string) (*database.InventoryCount, *responses.InternalResponse) {
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
	if !isCountOwnerOrAdmin(c, userID, callerRole) {
		return nil, forbidCountAction("enviar")
	}

	if resp := s.Repository.SubmitWithAdjustments(id, userID, s.AdjustmentsCreator); resp != nil {
		return nil, resp
	}
	updated, _ := s.Repository.GetByID(id)
	return updated, nil
}

// isCountOwnerOrAdmin returns true when callerUserID created the count or has the
// admin role. Used to gate destructive transitions (Submit, Cancel) per the
// W0 hostile review N2-1 finding.
func isCountOwnerOrAdmin(c *database.InventoryCount, callerUserID, callerRole string) bool {
	if c == nil {
		return false
	}
	if c.CreatedBy == callerUserID {
		return true
	}
	return strings.EqualFold(callerRole, "admin")
}

func forbidCountAction(verb string) *responses.InternalResponse {
	return &responses.InternalResponse{
		Message:    "Solo el creador del conteo o un administrador pueden " + verb + " este conteo",
		Handled:    true,
		StatusCode: responses.StatusForbidden,
	}
}
