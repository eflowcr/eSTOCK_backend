package controllers

import (
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ───────────────────────────────────────────────────────────────

type mockPresentationsRepoCtrl struct {
	all       []database.Presentations
	byID      map[string]*database.Presentations
	createErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockPresentationsRepoCtrl) GetAllPresentations() ([]database.Presentations, *responses.InternalResponse) {
	return m.all, nil
}

func (m *mockPresentationsRepoCtrl) GetPresentationByID(id string) (*database.Presentations, *responses.InternalResponse) {
	if m.byID != nil {
		if p, ok := m.byID[id]; ok {
			return p, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockPresentationsRepoCtrl) CreatePresentation(data *database.Presentations) *responses.InternalResponse {
	return m.createErr
}

func (m *mockPresentationsRepoCtrl) UpdatePresentation(id, name string) (*database.Presentations, *responses.InternalResponse) {
	if m.byID != nil {
		if p, ok := m.byID[id]; ok {
			return p, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockPresentationsRepoCtrl) DeletePresentation(id string) *responses.InternalResponse {
	return m.deleteErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newPresentationsController(repo *mockPresentationsRepoCtrl) *PresentationsController {
	svc := services.NewPresentationsService(repo)
	return NewPresentationsController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestPresentationsController_GetAllPresentations_Empty(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{all: []database.Presentations{}})
	w := performRequest(ctrl.GetAllPresentations, "GET", "/presentations", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationsController_GetAllPresentations_WithData(t *testing.T) {
	repo := &mockPresentationsRepoCtrl{
		all: []database.Presentations{{PresentationId: "p-1", Description: "Unit"}},
	}
	ctrl := newPresentationsController(repo)
	w := performRequest(ctrl.GetAllPresentations, "GET", "/presentations", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationsController_GetPresentationByID_Found(t *testing.T) {
	repo := &mockPresentationsRepoCtrl{
		byID: map[string]*database.Presentations{
			"p-1": {PresentationId: "p-1", Description: "Unit"},
		},
	}
	ctrl := newPresentationsController(repo)
	w := performRequest(ctrl.GetPresentationByID, "GET", "/presentations/p-1", nil, gin.Params{{Key: "id", Value: "p-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationsController_GetPresentationByID_NotFound(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{byID: map[string]*database.Presentations{}})
	w := performRequest(ctrl.GetPresentationByID, "GET", "/presentations/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPresentationsController_GetPresentationByID_MissingParam(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{})
	w := performRequest(ctrl.GetPresentationByID, "GET", "/presentations/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationsController_CreatePresentation_Success(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{})
	body := database.Presentations{PresentationId: "p-new", Description: "Box"}
	w := performRequest(ctrl.CreatePresentation, "POST", "/presentations", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPresentationsController_CreatePresentation_InvalidJSON(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{})
	w := performRequest(ctrl.CreatePresentation, "POST", "/presentations", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationsController_CreatePresentation_Conflict(t *testing.T) {
	repo := &mockPresentationsRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newPresentationsController(repo)
	body := database.Presentations{PresentationId: "p-dup", Description: "Duplicate"}
	w := performRequest(ctrl.CreatePresentation, "POST", "/presentations", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestPresentationsController_UpdatePresentation_Success(t *testing.T) {
	repo := &mockPresentationsRepoCtrl{
		byID: map[string]*database.Presentations{
			"p-1": {PresentationId: "p-1", Description: "Unit"},
		},
	}
	ctrl := newPresentationsController(repo)
	body := map[string]string{"description": "Updated Unit"}
	w := performRequest(ctrl.UpdatePresentation, "PUT", "/presentations/p-1", body, gin.Params{{Key: "id", Value: "p-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationsController_UpdatePresentation_NotFound(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{byID: map[string]*database.Presentations{}})
	body := map[string]string{"description": "Updated"}
	w := performRequest(ctrl.UpdatePresentation, "PUT", "/presentations/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPresentationsController_UpdatePresentation_MissingParam(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{})
	body := map[string]string{"description": "Updated"}
	w := performRequest(ctrl.UpdatePresentation, "PUT", "/presentations/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPresentationsController_DeletePresentation_Success(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{})
	w := performRequest(ctrl.DeletePresentation, "DELETE", "/presentations/p-1", nil, gin.Params{{Key: "id", Value: "p-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPresentationsController_DeletePresentation_Error(t *testing.T) {
	repo := &mockPresentationsRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newPresentationsController(repo)
	w := performRequest(ctrl.DeletePresentation, "DELETE", "/presentations/p-1", nil, gin.Params{{Key: "id", Value: "p-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPresentationsController_DeletePresentation_MissingParam(t *testing.T) {
	ctrl := newPresentationsController(&mockPresentationsRepoCtrl{})
	w := performRequest(ctrl.DeletePresentation, "DELETE", "/presentations/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
