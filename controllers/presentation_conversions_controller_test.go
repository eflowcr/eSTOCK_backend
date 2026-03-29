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

type mockPresentationConversionsRepoCtrl struct {
	list      []database.PresentationConversion
	byID      map[string]*database.PresentationConversion
	createErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockPresentationConversionsRepoCtrl) ListPresentationConversions() ([]database.PresentationConversion, *responses.InternalResponse) {
	return m.list, nil
}

func (m *mockPresentationConversionsRepoCtrl) ListPresentationConversionsAdmin() ([]database.PresentationConversion, *responses.InternalResponse) {
	return m.list, nil
}

func (m *mockPresentationConversionsRepoCtrl) GetPresentationConversionByID(id string) (*database.PresentationConversion, *responses.InternalResponse) {
	if m.byID != nil {
		if pc, ok := m.byID[id]; ok {
			return pc, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockPresentationConversionsRepoCtrl) GetPresentationConversionByFromAndTo(fromID, toID string) (*database.PresentationConversion, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockPresentationConversionsRepoCtrl) CreatePresentationConversion(req *requests.PresentationConversionCreate) (*database.PresentationConversion, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &database.PresentationConversion{
		ID:                     "pc-new",
		FromPresentationTypeID: req.FromPresentationTypeID,
		ToPresentationTypeID:   req.ToPresentationTypeID,
		ConversionFactor:       req.ConversionFactor,
	}, nil
}

func (m *mockPresentationConversionsRepoCtrl) UpdatePresentationConversion(id string, req *requests.PresentationConversionUpdate) (*database.PresentationConversion, *responses.InternalResponse) {
	if m.byID != nil {
		if pc, ok := m.byID[id]; ok {
			return pc, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockPresentationConversionsRepoCtrl) DeletePresentationConversion(id string) *responses.InternalResponse {
	return m.deleteErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newPresentationConversionsController(repo *mockPresentationConversionsRepoCtrl) *PresentationConversionsController {
	svc := services.NewPresentationConversionsService(repo)
	return NewPresentationConversionsController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestPresentationConversionsController_ListPresentationConversions_Empty(t *testing.T) {
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{list: []database.PresentationConversion{}})
	w := performRequest(ctrl.ListPresentationConversions, "GET", "/presentation-conversions", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationConversionsController_ListPresentationConversions_WithData(t *testing.T) {
	repo := &mockPresentationConversionsRepoCtrl{
		list: []database.PresentationConversion{{ID: "pc-1", ConversionFactor: 10.0}},
	}
	ctrl := newPresentationConversionsController(repo)
	w := performRequest(ctrl.ListPresentationConversions, "GET", "/presentation-conversions", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationConversionsController_ListPresentationConversionsAdmin_Success(t *testing.T) {
	repo := &mockPresentationConversionsRepoCtrl{
		list: []database.PresentationConversion{{ID: "pc-1", ConversionFactor: 10.0}},
	}
	ctrl := newPresentationConversionsController(repo)
	w := performRequest(ctrl.ListPresentationConversionsAdmin, "GET", "/presentation-conversions/admin", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationConversionsController_GetPresentationConversionByID_Found(t *testing.T) {
	repo := &mockPresentationConversionsRepoCtrl{
		byID: map[string]*database.PresentationConversion{
			"pc-1": {ID: "pc-1", ConversionFactor: 10.0},
		},
	}
	ctrl := newPresentationConversionsController(repo)
	w := performRequest(ctrl.GetPresentationConversionByID, "GET", "/presentation-conversions/pc-1", nil, gin.Params{{Key: "id", Value: "pc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationConversionsController_GetPresentationConversionByID_NotFound(t *testing.T) {
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{byID: map[string]*database.PresentationConversion{}})
	w := performRequest(ctrl.GetPresentationConversionByID, "GET", "/presentation-conversions/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPresentationConversionsController_GetPresentationConversionByID_MissingParam(t *testing.T) {
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{})
	w := performRequest(ctrl.GetPresentationConversionByID, "GET", "/presentation-conversions/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationConversionsController_CreatePresentationConversion_Success(t *testing.T) {
	isActive := true
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{})
	body := requests.PresentationConversionCreate{
		FromPresentationTypeID: "pt-1",
		ToPresentationTypeID:   "pt-2",
		ConversionFactor:       10.0,
		IsActive:               &isActive,
	}
	w := performRequest(ctrl.CreatePresentationConversion, "POST", "/presentation-conversions", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPresentationConversionsController_CreatePresentationConversion_InvalidJSON(t *testing.T) {
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{})
	w := performRequest(ctrl.CreatePresentationConversion, "POST", "/presentation-conversions", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationConversionsController_CreatePresentationConversion_Conflict(t *testing.T) {
	isActive := true
	repo := &mockPresentationConversionsRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "conversion already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newPresentationConversionsController(repo)
	body := requests.PresentationConversionCreate{
		FromPresentationTypeID: "pt-1",
		ToPresentationTypeID:   "pt-2",
		ConversionFactor:       10.0,
		IsActive:               &isActive,
	}
	w := performRequest(ctrl.CreatePresentationConversion, "POST", "/presentation-conversions", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPresentationConversionsController_UpdatePresentationConversion_Success(t *testing.T) {
	isActive := true
	repo := &mockPresentationConversionsRepoCtrl{
		byID: map[string]*database.PresentationConversion{
			"pc-1": {ID: "pc-1", ConversionFactor: 10.0},
		},
	}
	ctrl := newPresentationConversionsController(repo)
	body := requests.PresentationConversionUpdate{ConversionFactor: 20.0, IsActive: &isActive}
	w := performRequest(ctrl.UpdatePresentationConversion, "PUT", "/presentation-conversions/pc-1", body, gin.Params{{Key: "id", Value: "pc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationConversionsController_UpdatePresentationConversion_NotFound(t *testing.T) {
	isActive := true
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{byID: map[string]*database.PresentationConversion{}})
	body := requests.PresentationConversionUpdate{ConversionFactor: 20.0, IsActive: &isActive}
	w := performRequest(ctrl.UpdatePresentationConversion, "PUT", "/presentation-conversions/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPresentationConversionsController_UpdatePresentationConversion_MissingParam(t *testing.T) {
	isActive := true
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{})
	body := requests.PresentationConversionUpdate{ConversionFactor: 20.0, IsActive: &isActive}
	w := performRequest(ctrl.UpdatePresentationConversion, "PUT", "/presentation-conversions/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationConversionsController_DeletePresentationConversion_Success(t *testing.T) {
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{})
	w := performRequest(ctrl.DeletePresentationConversion, "DELETE", "/presentation-conversions/pc-1", nil, gin.Params{{Key: "id", Value: "pc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationConversionsController_DeletePresentationConversion_Error(t *testing.T) {
	repo := &mockPresentationConversionsRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newPresentationConversionsController(repo)
	w := performRequest(ctrl.DeletePresentationConversion, "DELETE", "/presentation-conversions/pc-1", nil, gin.Params{{Key: "id", Value: "pc-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPresentationConversionsController_DeletePresentationConversion_MissingParam(t *testing.T) {
	ctrl := newPresentationConversionsController(&mockPresentationConversionsRepoCtrl{})
	w := performRequest(ctrl.DeletePresentationConversion, "DELETE", "/presentation-conversions/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
