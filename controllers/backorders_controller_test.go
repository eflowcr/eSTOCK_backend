package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// stub repo
// ─────────────────────────────────────────────────────────────────────────────

type stubBORepo struct {
	listResult    *responses.BackorderListResponse
	listErr       *responses.InternalResponse
	getResult     *responses.BackorderResponse
	getErr        *responses.InternalResponse
	fulfillResult *responses.FulfillBackorderResult
	fulfillErr    *responses.InternalResponse
}

func (s *stubBORepo) List(_ string, _, _ *string, _, _ int) (*responses.BackorderListResponse, *responses.InternalResponse) {
	return s.listResult, s.listErr
}
func (s *stubBORepo) GetByID(_, _ string) (*responses.BackorderResponse, *responses.InternalResponse) {
	return s.getResult, s.getErr
}
func (s *stubBORepo) Fulfill(_, _, _ string) (*responses.FulfillBackorderResult, *responses.InternalResponse) {
	return s.fulfillResult, s.fulfillErr
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func boController(repo *stubBORepo) *BackordersController {
	svc := services.NewBackordersService(repo)
	return NewBackordersController(svc, "tenant-test")
}

func boGin(ctrl *BackordersController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(tools.ContextKeyUserID, "user-test")
		c.Next()
	})
	r.GET("/backorders", ctrl.List)
	r.GET("/backorders/:id", ctrl.GetByID)
	r.POST("/backorders/:id/fulfill", ctrl.Fulfill)
	return r
}

// ─────────────────────────────────────────────────────────────────────────────
// tests
// ─────────────────────────────────────────────────────────────────────────────

func TestBOController_List_OK(t *testing.T) {
	repo := &stubBORepo{listResult: &responses.BackorderListResponse{
		Items: []responses.BackorderResponse{{ID: "bo-1", Status: "pending", RemainingQty: 5}},
		Total: 1, Page: 1, Limit: 50,
	}}
	r := boGin(boController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/backorders", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body["data"])
}

func TestBOController_GetByID_OK(t *testing.T) {
	repo := &stubBORepo{getResult: &responses.BackorderResponse{ID: "bo-1", Status: "pending"}}
	r := boGin(boController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/backorders/bo-1", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestBOController_GetByID_NotFound(t *testing.T) {
	repo := &stubBORepo{getErr: &responses.InternalResponse{
		Message:    "Backorder no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}}
	r := boGin(boController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/backorders/nonexistent", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestBOController_Fulfill_OK(t *testing.T) {
	repo := &stubBORepo{fulfillResult: &responses.FulfillBackorderResult{
		Backorder:     &responses.BackorderResponse{ID: "bo-1"},
		PickingTaskID: "pick-xyz",
	}}
	r := boGin(boController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/backorders/bo-1/fulfill", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestBOController_Fulfill_NoStock(t *testing.T) {
	repo := &stubBORepo{fulfillErr: &responses.InternalResponse{
		Message:    "No hay stock disponible",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}}
	r := boGin(boController(repo))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/backorders/bo-1/fulfill", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
