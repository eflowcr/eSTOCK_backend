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

type mockSerialsRepoCtrl struct {
	byID      map[string]*database.Serial
	bySKU     map[string][]database.Serial
	createErr *responses.InternalResponse
	updateErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockSerialsRepoCtrl) GetSerialByID(id string) (*database.Serial, *responses.InternalResponse) {
	if m.byID != nil {
		if s, ok := m.byID[id]; ok {
			return s, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockSerialsRepoCtrl) GetSerialsBySKU(sku string) ([]database.Serial, *responses.InternalResponse) {
	if m.bySKU != nil {
		if serials, ok := m.bySKU[sku]; ok {
			return serials, nil
		}
	}
	return []database.Serial{}, nil
}

func (m *mockSerialsRepoCtrl) CreateSerial(data *requests.CreateSerialRequest) *responses.InternalResponse {
	return m.createErr
}

func (m *mockSerialsRepoCtrl) UpdateSerial(id string, data map[string]interface{}) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockSerialsRepoCtrl) DeleteSerial(id string) *responses.InternalResponse {
	return m.deleteErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newSerialsController(repo *mockSerialsRepoCtrl) *SerialsController {
	svc := services.NewSerialsService(repo)
	return NewSerialsController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestSerialsController_GetSerialByID_Found(t *testing.T) {
	repo := &mockSerialsRepoCtrl{
		byID: map[string]*database.Serial{
			"ser-1": {ID: "ser-1", SerialNumber: "SN-001", SKU: "SKU-001", Status: "available"},
		},
	}
	ctrl := newSerialsController(repo)
	w := performRequest(ctrl.GetSerialByID, "GET", "/serials/ser-1", nil, gin.Params{{Key: "id", Value: "ser-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSerialsController_GetSerialByID_NotFound(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{byID: map[string]*database.Serial{}})
	w := performRequest(ctrl.GetSerialByID, "GET", "/serials/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSerialsController_GetSerialByID_MissingParam(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.GetSerialByID, "GET", "/serials/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_GetSerialsBySKU_WithData(t *testing.T) {
	repo := &mockSerialsRepoCtrl{
		bySKU: map[string][]database.Serial{
			"SKU-001": {{ID: "ser-1", SerialNumber: "SN-001", SKU: "SKU-001", Status: "available"}},
		},
	}
	ctrl := newSerialsController(repo)
	w := performRequest(ctrl.GetSerialsBySKU, "GET", "/serials/SKU-001", nil, gin.Params{{Key: "sku", Value: "SKU-001"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSerialsController_GetSerialsBySKU_Empty(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.GetSerialsBySKU, "GET", "/serials/SKU-999", nil, gin.Params{{Key: "sku", Value: "SKU-999"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSerialsController_GetSerialsBySKU_MissingParam(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	// The controller reads sku via ctx.Param("sku") — empty string triggers 400
	w := performRequest(ctrl.GetSerialsBySKU, "GET", "/serials/", nil, gin.Params{{Key: "sku", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_CreateSerial_Success(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	body := requests.CreateSerialRequest{SerialNumber: "SN-001", SKU: "SKU-001"}
	w := performRequest(ctrl.CreateSerial, "POST", "/serials", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestSerialsController_CreateSerial_InvalidJSON(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.CreateSerial, "POST", "/serials", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_CreateSerial_Conflict(t *testing.T) {
	repo := &mockSerialsRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "serial number already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newSerialsController(repo)
	body := requests.CreateSerialRequest{SerialNumber: "SN-DUP", SKU: "SKU-001"}
	w := performRequest(ctrl.CreateSerial, "POST", "/serials", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestSerialsController_UpdateSerial_Success(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	body := map[string]interface{}{"status": "in_use"}
	w := performRequest(ctrl.UpdateSerial, "PUT", "/serials/ser-1", body, gin.Params{{Key: "id", Value: "ser-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSerialsController_UpdateSerial_MissingParam(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	body := map[string]interface{}{"status": "in_use"}
	w := performRequest(ctrl.UpdateSerial, "PUT", "/serials/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_UpdateSerial_NotFound(t *testing.T) {
	repo := &mockSerialsRepoCtrl{
		updateErr: &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound},
	}
	ctrl := newSerialsController(repo)
	body := map[string]interface{}{"status": "in_use"}
	w := performRequest(ctrl.UpdateSerial, "PUT", "/serials/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSerialsController_DeleteSerial_Success(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.DeleteSerial, "DELETE", "/serials/ser-1", nil, gin.Params{{Key: "id", Value: "ser-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSerialsController_DeleteSerial_MissingParam(t *testing.T) {
	ctrl := newSerialsController(&mockSerialsRepoCtrl{})
	w := performRequest(ctrl.DeleteSerial, "DELETE", "/serials/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSerialsController_DeleteSerial_Error(t *testing.T) {
	repo := &mockSerialsRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newSerialsController(repo)
	w := performRequest(ctrl.DeleteSerial, "DELETE", "/serials/ser-1", nil, gin.Params{{Key: "id", Value: "ser-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
