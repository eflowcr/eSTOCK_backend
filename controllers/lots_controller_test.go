package controllers

import (
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ───────────────────────────────────────────────────────────────

type mockLotsRepoCtrl struct {
	lots      []database.Lot
	bySKU     map[string][]database.Lot
	createErr *responses.InternalResponse
	updateErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockLotsRepoCtrl) GetAllLots() ([]database.Lot, *responses.InternalResponse) {
	return m.lots, nil
}

func (m *mockLotsRepoCtrl) GetLotsBySKU(sku *string) ([]database.Lot, *responses.InternalResponse) {
	if sku == nil || *sku == "" {
		return nil, nil
	}
	if m.bySKU != nil {
		if lots, ok := m.bySKU[*sku]; ok {
			return lots, nil
		}
	}
	return []database.Lot{}, nil
}

func (m *mockLotsRepoCtrl) CreateLot(data *requests.CreateLotRequest) *responses.InternalResponse {
	return m.createErr
}

func (m *mockLotsRepoCtrl) UpdateLot(id string, data map[string]interface{}) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockLotsRepoCtrl) DeleteLot(id string) *responses.InternalResponse {
	return m.deleteErr
}

func (m *mockLotsRepoCtrl) GetLotByID(id string) (*database.Lot, *responses.InternalResponse) {
	for i := range m.lots {
		if m.lots[i].ID == id {
			return &m.lots[i], nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockLotsRepoCtrl) GetLotTrace(_ string) (*responses.LotTraceResponse, *responses.InternalResponse) {
	return nil, nil
}

// ─── helper ──────────────────────────────────────────────────────────────────

func newLotsController(repo *mockLotsRepoCtrl) *LotsController {
	svc := services.NewLotsService(repo, nil)
	return NewLotsController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestLotsController_GetAllLots_Empty(t *testing.T) {
	ctrl := newLotsController(&mockLotsRepoCtrl{lots: []database.Lot{}})
	w := performRequest(ctrl.GetAllLots, "GET", "/lots", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLotsController_GetAllLots_WithData(t *testing.T) {
	repo := &mockLotsRepoCtrl{
		lots: []database.Lot{{ID: "lot-1", LotNumber: "L-001", SKU: "SKU1", Quantity: 10}},
	}
	ctrl := newLotsController(repo)
	w := performRequest(ctrl.GetAllLots, "GET", "/lots", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLotsController_GetLotsBySKU_Found(t *testing.T) {
	repo := &mockLotsRepoCtrl{
		bySKU: map[string][]database.Lot{
			"SKU1": {{ID: "lot-1", LotNumber: "L-001", SKU: "SKU1", Quantity: 10}},
		},
	}
	ctrl := newLotsController(repo)
	w := performRequest(ctrl.GetLotsBySKU, "GET", "/lots/SKU1", nil, gin.Params{{Key: "sku", Value: "SKU1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLotsController_GetLotsBySKU_Empty(t *testing.T) {
	ctrl := newLotsController(&mockLotsRepoCtrl{bySKU: map[string][]database.Lot{}})
	w := performRequest(ctrl.GetLotsBySKU, "GET", "/lots/NOSKU", nil, gin.Params{{Key: "sku", Value: "NOSKU"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLotsController_CreateLot_Success(t *testing.T) {
	ctrl := newLotsController(&mockLotsRepoCtrl{})
	body := requests.CreateLotRequest{
		LotNumber: "L-NEW",
		SKU:       "SKU1",
		Quantity:  5,
	}
	w := performRequest(ctrl.CreateLot, "POST", "/lots", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLotsController_CreateLot_InvalidJSON(t *testing.T) {
	ctrl := newLotsController(&mockLotsRepoCtrl{})
	w := performRequest(ctrl.CreateLot, "POST", "/lots", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLotsController_CreateLot_Conflict(t *testing.T) {
	repo := &mockLotsRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "lot number already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newLotsController(repo)
	body := requests.CreateLotRequest{
		LotNumber: "DUP-LOT",
		SKU:       "SKU1",
		Quantity:  5,
	}
	w := performRequest(ctrl.CreateLot, "POST", "/lots", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestLotsController_UpdateLot_Success(t *testing.T) {
	ctrl := newLotsController(&mockLotsRepoCtrl{})
	body := map[string]interface{}{"quantity": 20}
	w := performRequest(ctrl.UpdateLot, "PUT", "/lots/lot-1", body, gin.Params{{Key: "id", Value: "lot-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLotsController_UpdateLot_MissingParam(t *testing.T) {
	ctrl := newLotsController(&mockLotsRepoCtrl{})
	body := map[string]interface{}{"quantity": 20}
	w := performRequest(ctrl.UpdateLot, "PUT", "/lots/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLotsController_UpdateLot_NotFound(t *testing.T) {
	repo := &mockLotsRepoCtrl{
		updateErr: &responses.InternalResponse{
			Message:    "lot not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newLotsController(repo)
	body := map[string]interface{}{"quantity": 20}
	w := performRequest(ctrl.UpdateLot, "PUT", "/lots/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLotsController_DeleteLot_Success(t *testing.T) {
	ctrl := newLotsController(&mockLotsRepoCtrl{})
	w := performRequest(ctrl.DeleteLot, "DELETE", "/lots/lot-1", nil, gin.Params{{Key: "id", Value: "lot-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLotsController_DeleteLot_MissingParam(t *testing.T) {
	ctrl := newLotsController(&mockLotsRepoCtrl{})
	w := performRequest(ctrl.DeleteLot, "DELETE", "/lots/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLotsController_DeleteLot_NotFound(t *testing.T) {
	repo := &mockLotsRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "lot not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newLotsController(repo)
	w := performRequest(ctrl.DeleteLot, "DELETE", "/lots/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLotsController_DeleteLot_InternalError(t *testing.T) {
	repo := &mockLotsRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newLotsController(repo)
	w := performRequest(ctrl.DeleteLot, "DELETE", "/lots/lot-1", nil, gin.Params{{Key: "id", Value: "lot-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
