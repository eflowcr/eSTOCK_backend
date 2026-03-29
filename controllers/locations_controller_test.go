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

type mockLocationsRepoCtrl struct {
	locations []database.Location
	byID      map[string]*database.Location
	createErr *responses.InternalResponse
	updateErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockLocationsRepoCtrl) GetAllLocations() ([]database.Location, *responses.InternalResponse) {
	return m.locations, nil
}

func (m *mockLocationsRepoCtrl) GetLocationByID(id string) (*database.Location, *responses.InternalResponse) {
	if m.byID != nil {
		if l, ok := m.byID[id]; ok {
			return l, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockLocationsRepoCtrl) CreateLocation(loc *requests.Location) *responses.InternalResponse {
	return m.createErr
}

func (m *mockLocationsRepoCtrl) UpdateLocation(id string, data map[string]interface{}) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockLocationsRepoCtrl) DeleteLocation(id string) *responses.InternalResponse {
	return m.deleteErr
}

func (m *mockLocationsRepoCtrl) ImportLocationsFromExcel(fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	return []string{"LOC-001"}, nil, nil
}

func (m *mockLocationsRepoCtrl) ImportLocationsFromJSON(rows []requests.LocationImportRow) ([]string, []string, *responses.InternalResponse) {
	imported := make([]string, len(rows))
	for i, r := range rows {
		imported[i] = r.LocationCode
	}
	return imported, nil, nil
}

func (m *mockLocationsRepoCtrl) ValidateImportRows(rows []requests.LocationImportRow) ([]responses.LocationValidationResult, *responses.InternalResponse) {
	return []responses.LocationValidationResult{}, nil
}

func (m *mockLocationsRepoCtrl) ExportLocationsToExcel() ([]byte, *responses.InternalResponse) {
	return []byte("xlsx"), nil
}

func (m *mockLocationsRepoCtrl) GenerateImportTemplate(language string) ([]byte, error) {
	return []byte("tpl"), nil
}

// ─── helper ──────────────────────────────────────────────────────────────────

func newLocationsController(repo *mockLocationsRepoCtrl) *LocationsController {
	svc := services.NewLocationsService(repo)
	return NewLocationsController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestLocationsController_GetAllLocations_Empty(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{locations: []database.Location{}})
	w := performRequest(ctrl.GetAllLocations, "GET", "/locations", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationsController_GetAllLocations_WithData(t *testing.T) {
	repo := &mockLocationsRepoCtrl{
		locations: []database.Location{{ID: "loc-1", LocationCode: "A-01", Type: "shelf"}},
	}
	ctrl := newLocationsController(repo)
	w := performRequest(ctrl.GetAllLocations, "GET", "/locations", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationsController_GetLocationByID_Found(t *testing.T) {
	repo := &mockLocationsRepoCtrl{
		byID: map[string]*database.Location{
			"loc-1": {ID: "loc-1", LocationCode: "A-01", Type: "shelf"},
		},
	}
	ctrl := newLocationsController(repo)
	w := performRequest(ctrl.GetLocationByID, "GET", "/locations/loc-1", nil, gin.Params{{Key: "id", Value: "loc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationsController_GetLocationByID_NotFound(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{byID: map[string]*database.Location{}})
	w := performRequest(ctrl.GetLocationByID, "GET", "/locations/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLocationsController_GetLocationByID_MissingParam(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	w := performRequest(ctrl.GetLocationByID, "GET", "/locations/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLocationsController_CreateLocation_Success(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	body := requests.Location{LocationCode: "B-02", Type: "shelf"}
	w := performRequest(ctrl.CreateLocation, "POST", "/locations", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestLocationsController_CreateLocation_InvalidJSON(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	w := performRequest(ctrl.CreateLocation, "POST", "/locations", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLocationsController_CreateLocation_Conflict(t *testing.T) {
	repo := &mockLocationsRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "location_code already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newLocationsController(repo)
	body := requests.Location{LocationCode: "DUP-01", Type: "shelf"}
	w := performRequest(ctrl.CreateLocation, "POST", "/locations", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestLocationsController_UpdateLocation_Success(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	body := map[string]interface{}{"type": "rack"}
	w := performRequest(ctrl.UpdateLocation, "PUT", "/locations/loc-1", body, gin.Params{{Key: "id", Value: "loc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationsController_UpdateLocation_MissingParam(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	body := map[string]interface{}{"type": "rack"}
	w := performRequest(ctrl.UpdateLocation, "PUT", "/locations/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLocationsController_UpdateLocation_NotFound(t *testing.T) {
	repo := &mockLocationsRepoCtrl{
		updateErr: &responses.InternalResponse{
			Message:    "location not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newLocationsController(repo)
	body := map[string]interface{}{"type": "rack"}
	w := performRequest(ctrl.UpdateLocation, "PUT", "/locations/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLocationsController_DeleteLocation_Success(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	w := performRequest(ctrl.DeleteLocation, "DELETE", "/locations/loc-1", nil, gin.Params{{Key: "id", Value: "loc-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationsController_DeleteLocation_MissingParam(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	w := performRequest(ctrl.DeleteLocation, "DELETE", "/locations/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLocationsController_DeleteLocation_NotFound(t *testing.T) {
	repo := &mockLocationsRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "location not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newLocationsController(repo)
	w := performRequest(ctrl.DeleteLocation, "DELETE", "/locations/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLocationsController_ExportLocationsToExcel(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	w := performRequest(ctrl.ExportLocationsToExcel, "GET", "/locations/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationsController_ValidateImportRows_Success(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	rows := []requests.LocationImportRow{
		{LocationCode: "A-01", Type: "shelf"},
	}
	w := performRequest(ctrl.ValidateImportRows, "POST", "/locations/validate", rows, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationsController_ValidateImportRows_EmptyBody(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	w := performRequest(ctrl.ValidateImportRows, "POST", "/locations/validate", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLocationsController_ImportLocationsFromJSON_Success(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	rows := []requests.LocationImportRow{
		{LocationCode: "A-01", Type: "shelf"},
	}
	w := performRequest(ctrl.ImportLocationsFromJSON, "POST", "/locations/import/json", rows, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLocationsController_ImportLocationsFromJSON_EmptyBody(t *testing.T) {
	ctrl := newLocationsController(&mockLocationsRepoCtrl{})
	w := performRequest(ctrl.ImportLocationsFromJSON, "POST", "/locations/import/json", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
