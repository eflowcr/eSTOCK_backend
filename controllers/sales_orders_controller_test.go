package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// stubSalesOrdersRepo implements ports.SalesOrdersRepository for controller tests.
// ─────────────────────────────────────────────────────────────────────────────

type stubSalesOrdersRepo struct {
	createResult  *responses.SalesOrderResponse
	createErr     *responses.InternalResponse
	listResult    *responses.SalesOrderListResponse
	listErr       *responses.InternalResponse
	getResult     *responses.SalesOrderResponse
	getErr        *responses.InternalResponse
	updateResult  *responses.SalesOrderResponse
	updateErr     *responses.InternalResponse
	deleteErr     *responses.InternalResponse
	submitResult  *responses.SubmitSalesOrderResult
	submitErr     *responses.InternalResponse
	cancelErr     *responses.InternalResponse
	updatePickErr *responses.InternalResponse
}

func (s *stubSalesOrdersRepo) Create(_ string, _ string, _ *requests.CreateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	return s.createResult, s.createErr
}
func (s *stubSalesOrdersRepo) List(_ string, _, _, _, _, _ *string, _, _ int) (*responses.SalesOrderListResponse, *responses.InternalResponse) {
	return s.listResult, s.listErr
}
func (s *stubSalesOrdersRepo) GetByID(_ string, _ string) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	return s.getResult, s.getErr
}
func (s *stubSalesOrdersRepo) Update(_ string, _ string, _ *requests.UpdateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	return s.updateResult, s.updateErr
}
func (s *stubSalesOrdersRepo) SoftDelete(_ string, _ string) *responses.InternalResponse {
	return s.deleteErr
}
func (s *stubSalesOrdersRepo) Submit(_ string, _ string, _ string) (*responses.SubmitSalesOrderResult, *responses.InternalResponse) {
	return s.submitResult, s.submitErr
}
func (s *stubSalesOrdersRepo) Cancel(_ string, _ string, _ string) *responses.InternalResponse {
	return s.cancelErr
}
func (s *stubSalesOrdersRepo) UpdatePickedQty(_ string, _ map[string]float64) (string, *responses.InternalResponse) {
	return "", s.updatePickErr
}

// ─────────────────────────────────────────────────────────────────────────────
// test helpers
// ─────────────────────────────────────────────────────────────────────────────

func newSOCtrl(stub *stubSalesOrdersRepo) *SalesOrdersController {
	svc := services.NewSalesOrdersService(stub)
	return NewSalesOrdersController(svc, testJWTSecret, "tenant-1")
}

func setupSORouter(stub *stubSalesOrdersRepo) (*gin.Engine, *SalesOrdersController) {
	gin.SetMode(gin.TestMode)
	ctrl := newSOCtrl(stub)

	r := gin.New()
	so := r.Group("/api/sales-orders")
	so.GET("", ctrl.List)
	so.GET("/:id", ctrl.GetByID)
	so.POST("", ctrl.Create)
	so.PATCH("/:id", ctrl.Update)
	so.DELETE("/:id", ctrl.SoftDelete)
	so.PATCH("/:id/submit", ctrl.Submit)
	so.PATCH("/:id/cancel", ctrl.Cancel)
	return r, ctrl
}

func sampleSOResp(id string) *responses.SalesOrderResponse {
	now := time.Now()
	return &responses.SalesOrderResponse{
		ID:         id,
		TenantID:   "tenant-1",
		SONumber:   "SO-2026-0001",
		CustomerID: "cust-1",
		Status:     "draft",
		CreatedAt:  now,
		UpdatedAt:  now,
		Items:      []database.SalesOrderItem{},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — GET /api/sales-orders
// ─────────────────────────────────────────────────────────────────────────────

func TestSOController_List_200(t *testing.T) {
	stub := &stubSalesOrdersRepo{
		listResult: &responses.SalesOrderListResponse{
			Items:      []responses.SalesOrderListItem{{ID: "so-1", SONumber: "SO-2026-0001"}},
			Total:      1,
			Page:       1,
			Limit:      20,
			TotalPages: 1,
		},
	}
	r, _ := setupSORouter(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/sales-orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSOController_List_RepoError(t *testing.T) {
	stub := &stubSalesOrdersRepo{
		listErr: &responses.InternalResponse{Message: "DB error", StatusCode: responses.StatusInternalServerError},
	}
	r, _ := setupSORouter(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/sales-orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// any non-200 status is fine; just ensure it's not a panic
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — GET /api/sales-orders/:id
// ─────────────────────────────────────────────────────────────────────────────

func TestSOController_GetByID_200(t *testing.T) {
	stub := &stubSalesOrdersRepo{getResult: sampleSOResp("so-1")}
	r, _ := setupSORouter(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/sales-orders/so-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSOController_GetByID_404(t *testing.T) {
	stub := &stubSalesOrdersRepo{
		getErr: &responses.InternalResponse{Message: "not found", StatusCode: responses.StatusNotFound},
	}
	r, _ := setupSORouter(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/sales-orders/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — POST /api/sales-orders
// ─────────────────────────────────────────────────────────────────────────────

func TestSOController_Create_201(t *testing.T) {
	stub := &stubSalesOrdersRepo{createResult: sampleSOResp("so-new")}
	ctrl := newSOCtrl(stub)

	body := map[string]interface{}{
		"customer_id": "cust-1",
		"items": []map[string]interface{}{
			{"article_sku": "SKU-A", "expected_qty": 5},
		},
	}
	w := performRequestWithHeader(ctrl.Create, "POST", "/api/sales-orders", body, nil,
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestSOController_Create_400_InvalidJSON(t *testing.T) {
	stub := &stubSalesOrdersRepo{}
	r, _ := setupSORouter(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/sales-orders", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSOController_Create_400_MissingCustomerID(t *testing.T) {
	stub := &stubSalesOrdersRepo{}
	r, _ := setupSORouter(stub)

	body := `{"items":[{"article_sku":"SKU-A","expected_qty":5}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/sales-orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — PATCH /api/sales-orders/:id
// ─────────────────────────────────────────────────────────────────────────────

func TestSOController_Update_200(t *testing.T) {
	stub := &stubSalesOrdersRepo{updateResult: sampleSOResp("so-1")}
	r, _ := setupSORouter(stub)

	body := `{"notes":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/sales-orders/so-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSOController_Update_400_NotDraft(t *testing.T) {
	stub := &stubSalesOrdersRepo{
		updateErr: &responses.InternalResponse{Message: "Solo drafts", StatusCode: responses.StatusBadRequest},
	}
	r, _ := setupSORouter(stub)

	body := `{"notes":"updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/sales-orders/so-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — DELETE /api/sales-orders/:id
// ─────────────────────────────────────────────────────────────────────────────

func TestSOController_SoftDelete_200(t *testing.T) {
	stub := &stubSalesOrdersRepo{}
	r, _ := setupSORouter(stub)

	req := httptest.NewRequest(http.MethodDelete, "/api/sales-orders/so-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSOController_SoftDelete_400_NotDraft(t *testing.T) {
	stub := &stubSalesOrdersRepo{
		deleteErr: &responses.InternalResponse{Message: "Solo drafts", StatusCode: responses.StatusBadRequest},
	}
	r, _ := setupSORouter(stub)

	req := httptest.NewRequest(http.MethodDelete, "/api/sales-orders/so-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO2 — PATCH /api/sales-orders/:id/submit
// ─────────────────────────────────────────────────────────────────────────────

func TestSOController_Submit_200(t *testing.T) {
	ptID := "pt-xyz"
	so := sampleSOResp("so-1")
	so.Status = "submitted"
	so.PickingTaskID = &ptID
	stub := &stubSalesOrdersRepo{
		submitResult: &responses.SubmitSalesOrderResult{
			SalesOrder:    so,
			PickingTaskID: ptID,
		},
	}
	ctrl := newSOCtrl(stub)

	w := performRequestWithHeader(ctrl.Submit, "PATCH", "/api/sales-orders/so-1/submit", nil,
		gin.Params{{Key: "id", Value: "so-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body["result"].(map[string]interface{})["success"].(bool))
}

func TestSOController_Submit_400_NotDraft(t *testing.T) {
	stub := &stubSalesOrdersRepo{
		submitErr: &responses.InternalResponse{Message: "Solo drafts", StatusCode: responses.StatusBadRequest},
	}
	ctrl := newSOCtrl(stub)

	w := performRequestWithHeader(ctrl.Submit, "PATCH", "/api/sales-orders/so-1/submit", nil,
		gin.Params{{Key: "id", Value: "so-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO2 — PATCH /api/sales-orders/:id/cancel
// ─────────────────────────────────────────────────────────────────────────────

func TestSOController_Cancel_200(t *testing.T) {
	stub := &stubSalesOrdersRepo{}
	ctrl := newSOCtrl(stub)

	w := performRequestWithHeader(ctrl.Cancel, "PATCH", "/api/sales-orders/so-1/cancel", nil,
		gin.Params{{Key: "id", Value: "so-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSOController_Cancel_400_AlreadyCompleted(t *testing.T) {
	stub := &stubSalesOrdersRepo{
		cancelErr: &responses.InternalResponse{Message: "ya completada", StatusCode: responses.StatusBadRequest},
	}
	ctrl := newSOCtrl(stub)

	w := performRequestWithHeader(ctrl.Cancel, "PATCH", "/api/sales-orders/so-1/cancel", nil,
		gin.Params{{Key: "id", Value: "so-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// Route registration sanity — ensure no duplicate routes
// ─────────────────────────────────────────────────────────────────────────────

func TestSOController_RouteRegistration_NoPanic(t *testing.T) {
	// If duplicate routes are registered, gin panics. This test ensures no panic.
	assert.NotPanics(t, func() {
		gin.SetMode(gin.TestMode)
		stub := &stubSalesOrdersRepo{}
		svc := services.NewSalesOrdersService(stub)
		ctrl := NewSalesOrdersController(svc, "secret", "tenant-1")

		r := gin.New()
		so := r.Group("/api/sales-orders")
		so.GET("", ctrl.List)
		so.GET("/:id", ctrl.GetByID)
		so.POST("", ctrl.Create)
		so.PATCH("/:id", ctrl.Update)
		so.DELETE("/:id", ctrl.SoftDelete)
		so.PATCH("/:id/submit", ctrl.Submit)
		so.PATCH("/:id/cancel", ctrl.Cancel)
	})
}
