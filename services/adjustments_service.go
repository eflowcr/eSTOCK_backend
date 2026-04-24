package services

import (
	"context"
	"fmt"
	"math"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
)

type AdjustmentsService struct {
	Repository            ports.AdjustmentsRepository
	ReasonCodesRepository ports.AdjustmentReasonCodesRepository
	NotificationsSvc      *NotificationsService
}

func NewAdjustmentsService(repo ports.AdjustmentsRepository, reasonCodesRepo ports.AdjustmentReasonCodesRepository) *AdjustmentsService {
	return &AdjustmentsService{
		Repository:            repo,
		ReasonCodesRepository: reasonCodesRepo,
	}
}

// WithNotifications attaches an optional NotificationsService for count_reconcile alerts.
func (s *AdjustmentsService) WithNotifications(n *NotificationsService) *AdjustmentsService {
	s.NotificationsSvc = n
	return s
}

// GetAllAdjustments returns all adjustments (no tenant filter).
// internal use only — bypass tenant. Prefer ListByTenant in HTTP handlers.
func (s *AdjustmentsService) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	return s.Repository.GetAllAdjustments()
}

// ListByTenant returns adjustments scoped to a specific tenant (S2.5 M3.1).
func (s *AdjustmentsService) ListByTenant(tenantID string) ([]database.Adjustment, *responses.InternalResponse) {
	return s.Repository.GetAllForTenant(tenantID)
}

func (s *AdjustmentsService) GetAdjustmentByID(id string) (*database.Adjustment, *responses.InternalResponse) {
	return s.Repository.GetAdjustmentByID(id)
}

func (s *AdjustmentsService) GetAdjustmentDetails(id string) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	return s.Repository.GetAdjustmentDetails(id)
}

func (s *AdjustmentsService) CreateAdjustment(userId string, tenantID string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse) {
	// Quantity must be non-negative; direction is determined by adjustment_type or reason code.
	if adjustment.AdjustmentQuantity < 0 {
		return nil, &responses.InternalResponse{
			Message:    "adjustment quantity must be zero or positive; add or subtract is determined by the reason code",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	adjType := adjustment.AdjustmentType
	if adjType == "" {
		adjType = "increase"
	}

	var signedQuantity float64

	switch adjType {
	case "decrease":
		signedQuantity = -adjustment.AdjustmentQuantity
		// Validate that the decrease won't violate reserved_qty.
		inv, resp := s.Repository.GetInventoryForAdjustment(adjustment.SKU, adjustment.Location)
		if resp != nil {
			return nil, resp
		}
		newQty := inv.Quantity + signedQuantity
		if newQty < inv.ReservedQty {
			availableQty := inv.Quantity - inv.ReservedQty
			return nil, &responses.InternalResponse{
				Message: fmt.Sprintf(
					"no puede disminuir %.2f — disponible: %.2f (qty: %.2f, reservado: %.2f). Cancele los pickings activos antes de ajustar",
					math.Abs(signedQuantity), availableQty, inv.Quantity, inv.ReservedQty,
				),
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}

	case "count_reconcile":
		// Target qty (absolute physical count). Compute delta = target - current.
		inv, resp := s.Repository.GetInventoryForAdjustment(adjustment.SKU, adjustment.Location)
		if resp != nil {
			return nil, resp
		}
		signedQuantity = adjustment.AdjustmentQuantity - inv.Quantity
		newQty := adjustment.AdjustmentQuantity

		// Always notify admin that a physical count reconciliation occurred.
		if s.NotificationsSvc != nil {
			msg := fmt.Sprintf(
				"Conteo físico: SKU %s en %s ajustado a %.2f (anterior: %.2f)",
				adjustment.SKU, adjustment.Location, adjustment.AdjustmentQuantity, inv.Quantity,
			)
			if newQty < inv.ReservedQty {
				msg += fmt.Sprintf(
					". ⚠️ Reservas comprometidas: %.2f reservado, solo %.2f disponible tras ajuste",
					inv.ReservedQty, newQty,
				)
			}
			go func(svc *NotificationsService, u, m string) {
				_ = svc.Send(context.Background(), u, "count_reconcile",
					"Reconciliación de inventario", m, "adjustment", "")
			}(s.NotificationsSvc, userId, msg)
		}

	default: // "increase" and legacy (no type)
		signedQuantity = adjustment.AdjustmentQuantity
		if s.ReasonCodesRepository != nil {
			reasonCode, resp := s.ReasonCodesRepository.GetAdjustmentReasonCodeByCode(adjustment.Reason)
			if resp != nil {
				return nil, resp
			}
			if reasonCode == nil {
				return nil, &responses.InternalResponse{
					Message:    "invalid or inactive reason code for adjustment",
					Handled:    true,
					StatusCode: responses.StatusBadRequest,
				}
			}
			if reasonCode.Direction == "outbound" {
				signedQuantity = -adjustment.AdjustmentQuantity
			}
		}
	}

	req := adjustment
	req.AdjustmentQuantity = signedQuantity
	req.AdjustmentType = adjType
	return s.Repository.CreateAdjustment(userId, tenantID, req)
}

func (s *AdjustmentsService) ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse) {
	return s.Repository.ExportAdjustmentsToExcel()
}
