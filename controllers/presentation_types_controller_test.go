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

type mockPresentationTypesRepoCtrl struct {
	list      []database.PresentationType
	byID      map[string]*database.PresentationType
	createErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockPresentationTypesRepoCtrl) ListPresentationTypes() ([]database.PresentationType, *responses.InternalResponse) {
	return m.list, nil
}

func (m *mockPresentationTypesRepoCtrl) ListPresentationTypesAdmin() ([]database.PresentationType, *responses.InternalResponse) {
	return m.list, nil
}

func (m *mockPresentationTypesRepoCtrl) GetPresentationTypeByID(id string) (*database.PresentationType, *responses.InternalResponse) {
	if m.byID != nil {
		if pt, ok := m.byID[id]; ok {
			return pt, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockPresentationTypesRepoCtrl) GetPresentationTypeByCode(code string) (*database.PresentationType, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockPresentationTypesRepoCtrl) CreatePresentationType(req *requests.PresentationTypeCreate) (*database.PresentationType, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &database.PresentationType{ID: "pt-new", Code: req.Code, Name: req.Name}, nil
}

func (m *mockPresentationTypesRepoCtrl) UpdatePresentationType(id string, req *requests.PresentationTypeUpdate) (*database.PresentationType, *responses.InternalResponse) {
	if m.byID != nil {
		if pt, ok := m.byID[id]; ok {
			return pt, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockPresentationTypesRepoCtrl) DeletePresentationType(id string) *responses.InternalResponse {
	return m.deleteErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newPresentationTypesController(repo *mockPresentationTypesRepoCtrl) *PresentationTypesController {
	svc := services.NewPresentationTypesService(repo)
	return NewPresentationTypesController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestPresentationTypesController_ListPresentationTypes_Empty(t *testing.T) {
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{list: []database.PresentationType{}})
	w := performRequest(ctrl.ListPresentationTypes, "GET", "/presentation-types", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationTypesController_ListPresentationTypes_WithData(t *testing.T) {
	repo := &mockPresentationTypesRepoCtrl{
		list: []database.PresentationType{{ID: "pt-1", Code: "UNIT", Name: "Unit"}},
	}
	ctrl := newPresentationTypesController(repo)
	w := performRequest(ctrl.ListPresentationTypes, "GET", "/presentation-types", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationTypesController_ListPresentationTypesAdmin_Success(t *testing.T) {
	repo := &mockPresentationTypesRepoCtrl{
		list: []database.PresentationType{{ID: "pt-1", Code: "UNIT", Name: "Unit"}},
	}
	ctrl := newPresentationTypesController(repo)
	w := performRequest(ctrl.ListPresentationTypesAdmin, "GET", "/presentation-types/admin", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationTypesController_GetPresentationTypeByID_Found(t *testing.T) {
	repo := &mockPresentationTypesRepoCtrl{
		byID: map[string]*database.PresentationType{
			"pt-1": {ID: "pt-1", Code: "UNIT", Name: "Unit"},
		},
	}
	ctrl := newPresentationTypesController(repo)
	w := performRequest(ctrl.GetPresentationTypeByID, "GET", "/presentation-types/pt-1", nil, gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationTypesController_GetPresentationTypeByID_NotFound(t *testing.T) {
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{byID: map[string]*database.PresentationType{}})
	w := performRequest(ctrl.GetPresentationTypeByID, "GET", "/presentation-types/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPresentationTypesController_GetPresentationTypeByID_MissingParam(t *testing.T) {
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{})
	w := performRequest(ctrl.GetPresentationTypeByID, "GET", "/presentation-types/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationTypesController_CreatePresentationType_Success(t *testing.T) {
	isActive := true
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{})
	body := requests.PresentationTypeCreate{Code: "BOX", Name: "Box", IsActive: &isActive}
	w := performRequest(ctrl.CreatePresentationType, "POST", "/presentation-types", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPresentationTypesController_CreatePresentationType_InvalidJSON(t *testing.T) {
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{})
	w := performRequest(ctrl.CreatePresentationType, "POST", "/presentation-types", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationTypesController_CreatePresentationType_Conflict(t *testing.T) {
	isActive := true
	repo := &mockPresentationTypesRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "code already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newPresentationTypesController(repo)
	body := requests.PresentationTypeCreate{Code: "DUP", Name: "Duplicate", IsActive: &isActive}
	w := performRequest(ctrl.CreatePresentationType, "POST", "/presentation-types", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPresentationTypesController_UpdatePresentationType_Success(t *testing.T) {
	isActive := true
	repo := &mockPresentationTypesRepoCtrl{
		byID: map[string]*database.PresentationType{
			"pt-1": {ID: "pt-1", Code: "UNIT", Name: "Unit"},
		},
	}
	ctrl := newPresentationTypesController(repo)
	body := requests.PresentationTypeUpdate{Code: "UNIT", Name: "Updated Unit", IsActive: &isActive}
	w := performRequest(ctrl.UpdatePresentationType, "PUT", "/presentation-types/pt-1", body, gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationTypesController_UpdatePresentationType_NotFound(t *testing.T) {
	isActive := true
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{byID: map[string]*database.PresentationType{}})
	body := requests.PresentationTypeUpdate{Code: "UNIT", Name: "Updated", IsActive: &isActive}
	w := performRequest(ctrl.UpdatePresentationType, "PUT", "/presentation-types/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPresentationTypesController_UpdatePresentationType_MissingParam(t *testing.T) {
	isActive := true
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{})
	body := requests.PresentationTypeUpdate{Code: "UNIT", Name: "Updated", IsActive: &isActive}
	w := performRequest(ctrl.UpdatePresentationType, "PUT", "/presentation-types/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationTypesController_DeletePresentationType_Success(t *testing.T) {
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{})
	w := performRequest(ctrl.DeletePresentationType, "DELETE", "/presentation-types/pt-1", nil, gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationTypesController_DeletePresentationType_Error(t *testing.T) {
	repo := &mockPresentationTypesRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newPresentationTypesController(repo)
	w := performRequest(ctrl.DeletePresentationType, "DELETE", "/presentation-types/pt-1", nil, gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPresentationTypesController_DeletePresentationType_MissingParam(t *testing.T) {
	ctrl := newPresentationTypesController(&mockPresentationTypesRepoCtrl{})
	w := performRequest(ctrl.DeletePresentationType, "DELETE", "/presentation-types/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
