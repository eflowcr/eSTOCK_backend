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

type mockLocationTypesRepoCtrl struct {
	list      []database.LocationType
	byID      map[string]*database.LocationType
	createErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockLocationTypesRepoCtrl) ListLocationTypes() ([]database.LocationType, *responses.InternalResponse) {
	return m.list, nil
}

func (m *mockLocationTypesRepoCtrl) ListLocationTypesAdmin() ([]database.LocationType, *responses.InternalResponse) {
	return m.list, nil
}

func (m *mockLocationTypesRepoCtrl) GetLocationTypeByID(id string) (*database.LocationType, *responses.InternalResponse) {
	if m.byID != nil {
		if lt, ok := m.byID[id]; ok {
			return lt, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockLocationTypesRepoCtrl) GetLocationTypeByCode(code string) (*database.LocationType, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockLocationTypesRepoCtrl) CreateLocationType(req *requests.LocationTypeCreate) (*database.LocationType, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &database.LocationType{ID: "lt-new", Code: req.Code, Name: req.Name}, nil
}

func (m *mockLocationTypesRepoCtrl) UpdateLocationType(id string, req *requests.LocationTypeUpdate) (*database.LocationType, *responses.InternalResponse) {
	if m.byID != nil {
		if lt, ok := m.byID[id]; ok {
			return lt, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockLocationTypesRepoCtrl) DeleteLocationType(id string) *responses.InternalResponse {
	return m.deleteErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newLocationTypesController(repo *mockLocationTypesRepoCtrl) *LocationTypesController {
	svc := services.NewLocationTypesService(repo)
	return NewLocationTypesController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestLocationTypesController_ListLocationTypes_Empty(t *testing.T) {
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{list: []database.LocationType{}})
	w := performRequest(ctrl.ListLocationTypes, "GET", "/location-types", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationTypesController_ListLocationTypes_WithData(t *testing.T) {
	repo := &mockLocationTypesRepoCtrl{
		list: []database.LocationType{{ID: "lt-1", Code: "RACK", Name: "Rack"}},
	}
	ctrl := newLocationTypesController(repo)
	w := performRequest(ctrl.ListLocationTypes, "GET", "/location-types", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationTypesController_ListLocationTypesAdmin_Success(t *testing.T) {
	repo := &mockLocationTypesRepoCtrl{
		list: []database.LocationType{{ID: "lt-1", Code: "RACK", Name: "Rack"}},
	}
	ctrl := newLocationTypesController(repo)
	w := performRequest(ctrl.ListLocationTypesAdmin, "GET", "/location-types/admin", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationTypesController_GetLocationTypeByID_Found(t *testing.T) {
	repo := &mockLocationTypesRepoCtrl{
		byID: map[string]*database.LocationType{
			"lt-1": {ID: "lt-1", Code: "RACK", Name: "Rack"},
		},
	}
	ctrl := newLocationTypesController(repo)
	w := performRequest(ctrl.GetLocationTypeByID, "GET", "/location-types/lt-1", nil, gin.Params{{Key: "id", Value: "lt-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationTypesController_GetLocationTypeByID_NotFound(t *testing.T) {
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{byID: map[string]*database.LocationType{}})
	w := performRequest(ctrl.GetLocationTypeByID, "GET", "/location-types/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLocationTypesController_GetLocationTypeByID_MissingParam(t *testing.T) {
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{})
	w := performRequest(ctrl.GetLocationTypeByID, "GET", "/location-types/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLocationTypesController_CreateLocationType_Success(t *testing.T) {
	isActive := true
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{})
	body := requests.LocationTypeCreate{Code: "SHELF", Name: "Shelf", IsActive: &isActive}
	w := performRequest(ctrl.CreateLocationType, "POST", "/location-types", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLocationTypesController_CreateLocationType_InvalidJSON(t *testing.T) {
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{})
	w := performRequest(ctrl.CreateLocationType, "POST", "/location-types", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLocationTypesController_CreateLocationType_Conflict(t *testing.T) {
	isActive := true
	repo := &mockLocationTypesRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "code already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newLocationTypesController(repo)
	body := requests.LocationTypeCreate{Code: "DUP", Name: "Duplicate", IsActive: &isActive}
	w := performRequest(ctrl.CreateLocationType, "POST", "/location-types", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestLocationTypesController_UpdateLocationType_Success(t *testing.T) {
	isActive := true
	repo := &mockLocationTypesRepoCtrl{
		byID: map[string]*database.LocationType{
			"lt-1": {ID: "lt-1", Code: "RACK", Name: "Rack"},
		},
	}
	ctrl := newLocationTypesController(repo)
	body := requests.LocationTypeUpdate{Code: "RACK", Name: "Updated Rack", IsActive: &isActive}
	w := performRequest(ctrl.UpdateLocationType, "PUT", "/location-types/lt-1", body, gin.Params{{Key: "id", Value: "lt-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationTypesController_UpdateLocationType_NotFound(t *testing.T) {
	isActive := true
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{byID: map[string]*database.LocationType{}})
	body := requests.LocationTypeUpdate{Code: "RACK", Name: "Updated", IsActive: &isActive}
	w := performRequest(ctrl.UpdateLocationType, "PUT", "/location-types/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLocationTypesController_UpdateLocationType_MissingParam(t *testing.T) {
	isActive := true
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{})
	body := requests.LocationTypeUpdate{Code: "RACK", Name: "Updated", IsActive: &isActive}
	w := performRequest(ctrl.UpdateLocationType, "PUT", "/location-types/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLocationTypesController_DeleteLocationType_Success(t *testing.T) {
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{})
	w := performRequest(ctrl.DeleteLocationType, "DELETE", "/location-types/lt-1", nil, gin.Params{{Key: "id", Value: "lt-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationTypesController_DeleteLocationType_Error(t *testing.T) {
	repo := &mockLocationTypesRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newLocationTypesController(repo)
	w := performRequest(ctrl.DeleteLocationType, "DELETE", "/location-types/lt-1", nil, gin.Params{{Key: "id", Value: "lt-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLocationTypesController_DeleteLocationType_MissingParam(t *testing.T) {
	ctrl := newLocationTypesController(&mockLocationTypesRepoCtrl{})
	w := performRequest(ctrl.DeleteLocationType, "DELETE", "/location-types/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
