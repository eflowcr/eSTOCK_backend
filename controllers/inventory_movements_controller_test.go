package controllers

import (
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ───────────────────────────────────────────────────────────────

type mockInventoryMovementsRepoCtrl struct {
	movements []database.InventoryMovement
	listErr   *responses.InternalResponse
}

func (m *mockInventoryMovementsRepoCtrl) GetAllInventoryMovements(sku string) ([]database.InventoryMovement, *responses.InternalResponse) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.movements, nil
}

func (m *mockInventoryMovementsRepoCtrl) ListMovements(_ ports.MovementsFilter) ([]database.InventoryMovement, *responses.InternalResponse) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.movements, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newInventoryMovementsController(repo *mockInventoryMovementsRepoCtrl) *InventoryMovementsController {
	svc := services.NewInventoryMovementsService(repo)
	return NewInventoryMovementsController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestInventoryMovementsController_GetAllInventoryMovements_Empty(t *testing.T) {
	ctrl := newInventoryMovementsController(&mockInventoryMovementsRepoCtrl{movements: []database.InventoryMovement{}})
	w := performRequest(ctrl.GetAllInventoryMovements, "GET", "/inventory-movements/SKU001", nil, gin.Params{{Key: "sku", Value: "SKU001"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryMovementsController_GetAllInventoryMovements_WithData(t *testing.T) {
	reason := "initial stock"
	repo := &mockInventoryMovementsRepoCtrl{
		movements: []database.InventoryMovement{
			{
				ID:             "mv-1",
				SKU:            "SKU001",
				Location:       "A-01",
				MovementType:   "IN",
				Quantity:       100.0,
				RemainingStock: 100.0,
				Reason:         &reason,
				CreatedBy:      "user-1",
			},
		},
	}
	ctrl := newInventoryMovementsController(repo)
	w := performRequest(ctrl.GetAllInventoryMovements, "GET", "/inventory-movements/SKU001", nil, gin.Params{{Key: "sku", Value: "SKU001"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryMovementsController_GetAllInventoryMovements_NoSKUParam(t *testing.T) {
	repo := &mockInventoryMovementsRepoCtrl{
		movements: []database.InventoryMovement{},
	}
	ctrl := newInventoryMovementsController(repo)
	// sku param empty — controller still calls the service with empty string, returns empty list → 200
	w := performRequest(ctrl.GetAllInventoryMovements, "GET", "/inventory-movements/", nil, gin.Params{{Key: "sku", Value: ""}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryMovementsController_GetAllInventoryMovements_Error(t *testing.T) {
	repo := &mockInventoryMovementsRepoCtrl{
		listErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newInventoryMovementsController(repo)
	w := performRequest(ctrl.GetAllInventoryMovements, "GET", "/inventory-movements/SKU001", nil, gin.Params{{Key: "sku", Value: "SKU001"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
