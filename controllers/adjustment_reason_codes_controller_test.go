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

type mockAdjustmentReasonCodesRepoCtrl struct {
	list      []database.AdjustmentReasonCode
	listAdmin []database.AdjustmentReasonCode
	byID      map[string]*database.AdjustmentReasonCode
	byCode    map[string]*database.AdjustmentReasonCode
	createErr *responses.InternalResponse
	updateErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockAdjustmentReasonCodesRepoCtrl) ListAdjustmentReasonCodes() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return m.list, nil
}

func (m *mockAdjustmentReasonCodesRepoCtrl) ListAdjustmentReasonCodesAdmin() ([]database.AdjustmentReasonCode, *responses.InternalResponse) {
	return m.listAdmin, nil
}

func (m *mockAdjustmentReasonCodesRepoCtrl) GetAdjustmentReasonCodeByID(id string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.byID != nil {
		if rc, ok := m.byID[id]; ok {
			return rc, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockAdjustmentReasonCodesRepoCtrl) GetAdjustmentReasonCodeByCode(code string) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.byCode != nil {
		if rc, ok := m.byCode[code]; ok {
			return rc, nil
		}
	}
	return nil, nil
}

func (m *mockAdjustmentReasonCodesRepoCtrl) CreateAdjustmentReasonCode(req *requests.AdjustmentReasonCodeCreate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &database.AdjustmentReasonCode{ID: "rc-new", Code: req.Code, Name: req.Name, Direction: req.Direction}, nil
}

func (m *mockAdjustmentReasonCodesRepoCtrl) UpdateAdjustmentReasonCode(id string, req *requests.AdjustmentReasonCodeUpdate) (*database.AdjustmentReasonCode, *responses.InternalResponse) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.byID != nil {
		if rc, ok := m.byID[id]; ok {
			return rc, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockAdjustmentReasonCodesRepoCtrl) DeleteAdjustmentReasonCode(id string) *responses.InternalResponse {
	return m.deleteErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newAdjustmentReasonCodesController(repo *mockAdjustmentReasonCodesRepoCtrl) *AdjustmentReasonCodesController {
	svc := services.NewAdjustmentReasonCodesService(repo)
	return NewAdjustmentReasonCodesController(*svc)
}

func sampleRC() database.AdjustmentReasonCode {
	return database.AdjustmentReasonCode{ID: "rc-1", Code: "INBOUND", Name: "Inbound Stock", Direction: "inbound", IsActive: true}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestAdjustmentReasonCodesController_ListAdjustmentReasonCodes_Empty(t *testing.T) {
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{list: []database.AdjustmentReasonCode{}})
	w := performRequest(ctrl.ListAdjustmentReasonCodes, "GET", "/adjustment-reason-codes", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentReasonCodesController_ListAdjustmentReasonCodes_WithData(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepoCtrl{list: []database.AdjustmentReasonCode{sampleRC()}}
	ctrl := newAdjustmentReasonCodesController(repo)
	w := performRequest(ctrl.ListAdjustmentReasonCodes, "GET", "/adjustment-reason-codes", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentReasonCodesController_ListAdjustmentReasonCodesAdmin_WithData(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepoCtrl{listAdmin: []database.AdjustmentReasonCode{sampleRC()}}
	ctrl := newAdjustmentReasonCodesController(repo)
	w := performRequest(ctrl.ListAdjustmentReasonCodesAdmin, "GET", "/adjustment-reason-codes/admin", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentReasonCodesController_GetAdjustmentReasonCodeByID_Found(t *testing.T) {
	rc := sampleRC()
	repo := &mockAdjustmentReasonCodesRepoCtrl{byID: map[string]*database.AdjustmentReasonCode{"rc-1": &rc}}
	ctrl := newAdjustmentReasonCodesController(repo)
	w := performRequest(ctrl.GetAdjustmentReasonCodeByID, "GET", "/adjustment-reason-codes/rc-1", nil, gin.Params{{Key: "id", Value: "rc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentReasonCodesController_GetAdjustmentReasonCodeByID_NotFound(t *testing.T) {
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{byID: map[string]*database.AdjustmentReasonCode{}})
	w := performRequest(ctrl.GetAdjustmentReasonCodeByID, "GET", "/adjustment-reason-codes/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdjustmentReasonCodesController_GetAdjustmentReasonCodeByID_MissingParam(t *testing.T) {
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{})
	w := performRequest(ctrl.GetAdjustmentReasonCodeByID, "GET", "/adjustment-reason-codes/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustmentReasonCodesController_CreateAdjustmentReasonCode_Success(t *testing.T) {
	isActive := true
	body := requests.AdjustmentReasonCodeCreate{Code: "NEW-CODE", Name: "New Code", Direction: "inbound", IsActive: &isActive}
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{})
	w := performRequest(ctrl.CreateAdjustmentReasonCode, "POST", "/adjustment-reason-codes", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAdjustmentReasonCodesController_CreateAdjustmentReasonCode_InvalidJSON(t *testing.T) {
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{})
	w := performRequest(ctrl.CreateAdjustmentReasonCode, "POST", "/adjustment-reason-codes", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustmentReasonCodesController_CreateAdjustmentReasonCode_Conflict(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "code already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newAdjustmentReasonCodesController(repo)
	isActive := true
	body := requests.AdjustmentReasonCodeCreate{Code: "DUP", Name: "Duplicate", Direction: "inbound", IsActive: &isActive}
	w := performRequest(ctrl.CreateAdjustmentReasonCode, "POST", "/adjustment-reason-codes", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAdjustmentReasonCodesController_UpdateAdjustmentReasonCode_Success(t *testing.T) {
	rc := sampleRC()
	repo := &mockAdjustmentReasonCodesRepoCtrl{byID: map[string]*database.AdjustmentReasonCode{"rc-1": &rc}}
	ctrl := newAdjustmentReasonCodesController(repo)
	isActive := true
	body := requests.AdjustmentReasonCodeUpdate{Code: "INBOUND", Name: "Updated", Direction: "inbound", IsActive: &isActive}
	w := performRequest(ctrl.UpdateAdjustmentReasonCode, "PUT", "/adjustment-reason-codes/rc-1", body, gin.Params{{Key: "id", Value: "rc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentReasonCodesController_UpdateAdjustmentReasonCode_MissingParam(t *testing.T) {
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{})
	isActive := true
	body := requests.AdjustmentReasonCodeUpdate{Code: "INBOUND", Name: "Updated", Direction: "inbound", IsActive: &isActive}
	w := performRequest(ctrl.UpdateAdjustmentReasonCode, "PUT", "/adjustment-reason-codes/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustmentReasonCodesController_UpdateAdjustmentReasonCode_InvalidJSON(t *testing.T) {
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{})
	w := performRequest(ctrl.UpdateAdjustmentReasonCode, "PUT", "/adjustment-reason-codes/rc-1", nil, gin.Params{{Key: "id", Value: "rc-1"}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustmentReasonCodesController_UpdateAdjustmentReasonCode_NotFound(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepoCtrl{
		updateErr: &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound},
	}
	ctrl := newAdjustmentReasonCodesController(repo)
	isActive := true
	body := requests.AdjustmentReasonCodeUpdate{Code: "INBOUND", Name: "Updated", Direction: "inbound", IsActive: &isActive}
	w := performRequest(ctrl.UpdateAdjustmentReasonCode, "PUT", "/adjustment-reason-codes/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdjustmentReasonCodesController_DeleteAdjustmentReasonCode_Success(t *testing.T) {
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{})
	w := performRequest(ctrl.DeleteAdjustmentReasonCode, "DELETE", "/adjustment-reason-codes/rc-1", nil, gin.Params{{Key: "id", Value: "rc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdjustmentReasonCodesController_DeleteAdjustmentReasonCode_MissingParam(t *testing.T) {
	ctrl := newAdjustmentReasonCodesController(&mockAdjustmentReasonCodesRepoCtrl{})
	w := performRequest(ctrl.DeleteAdjustmentReasonCode, "DELETE", "/adjustment-reason-codes/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustmentReasonCodesController_DeleteAdjustmentReasonCode_Error(t *testing.T) {
	repo := &mockAdjustmentReasonCodesRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newAdjustmentReasonCodesController(repo)
	w := performRequest(ctrl.DeleteAdjustmentReasonCode, "DELETE", "/adjustment-reason-codes/rc-1", nil, gin.Params{{Key: "id", Value: "rc-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
