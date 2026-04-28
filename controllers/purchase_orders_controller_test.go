package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock repository for controller tests
// ─────────────────────────────────────────────────────────────────────────────

type mockPOCtrlRepo struct {
	created     *responses.PurchaseOrderView
	createErr   *responses.InternalResponse
	byID        map[string]*responses.PurchaseOrderView
	listed      []responses.PurchaseOrderView
	listErr     *responses.InternalResponse
	updated     *responses.PurchaseOrderView
	updateErr   *responses.InternalResponse
	softDelErr  *responses.InternalResponse
	submitView  *responses.PurchaseOrderView
	submitRTID  string
	submitErr   *responses.InternalResponse
	cancelView  *responses.PurchaseOrderView
	cancelErr   *responses.InternalResponse
}

func (m *mockPOCtrlRepo) Create(tenantID, createdBy string, req *requests.CreatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return m.created, m.createErr
}
func (m *mockPOCtrlRepo) GetByID(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	if m.byID != nil {
		if v, ok := m.byID[id]; ok {
			return v, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}
func (m *mockPOCtrlRepo) List(tenantID string, status, supplierID, search *string, from, to *string, limit, offset int) ([]responses.PurchaseOrderView, *responses.InternalResponse) {
	return m.listed, m.listErr
}
func (m *mockPOCtrlRepo) Update(id, tenantID string, req *requests.UpdatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return m.updated, m.updateErr
}
func (m *mockPOCtrlRepo) SoftDelete(id, tenantID string) *responses.InternalResponse {
	return m.softDelErr
}
func (m *mockPOCtrlRepo) Submit(id, tenantID, userID string) (*responses.PurchaseOrderView, string, *responses.InternalResponse) {
	return m.submitView, m.submitRTID, m.submitErr
}
func (m *mockPOCtrlRepo) Cancel(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return m.cancelView, m.cancelErr
}
func (m *mockPOCtrlRepo) UpdateReceivedQty(purchaseOrderID string, updates []database.PurchaseOrderItemQtyUpdate) *responses.InternalResponse {
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

const ctrlTenantID = "00000000-0000-0000-0000-000000000001"

func newPOTestRouter(repo *mockPOCtrlRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := services.NewPurchaseOrdersService(repo)
	ctrl := NewPurchaseOrdersController(svc, ctrlTenantID)

	// Inject a fake user_id into context (simulates JWTAuthMiddleware).
	injectUser := func(c *gin.Context) {
		c.Set(tools.ContextKeyUserID, "test-user")
		c.Next()
	}

	api := r.Group("/api")
	po := api.Group("/purchase-orders")
	po.Use(injectUser)
	po.GET("/", ctrl.List)
	po.GET("/:id", ctrl.GetByID)
	po.POST("/", ctrl.Create)
	po.PATCH("/:id", ctrl.Update)
	po.DELETE("/:id", ctrl.Delete)
	po.PATCH("/:id/submit", ctrl.Submit)
	po.PATCH("/:id/cancel", ctrl.Cancel)
	return r
}

func samplePOCtrlView(id, status string) *responses.PurchaseOrderView {
	return &responses.PurchaseOrderView{
		ID:         id,
		PONumber:   "PO-2026-0001",
		SupplierID: "supplier-1",
		Status:     status,
		TenantID:   ctrlTenantID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PO1 — CRUD controller tests
// ─────────────────────────────────────────────────────────────────────────────

func TestPOController_Create_Returns201(t *testing.T) {
	repo := &mockPOCtrlRepo{created: samplePOCtrlView("po-1", "draft")}
	r := newPOTestRouter(repo)

	body := map[string]interface{}{
		"supplier_id": "supplier-1",
		"items": []map[string]interface{}{
			{"article_sku": "SKU-001", "expected_qty": 10},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/purchase-orders/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPOController_Create_Returns400_MissingSupplier(t *testing.T) {
	repo := &mockPOCtrlRepo{}
	r := newPOTestRouter(repo)

	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"article_sku": "SKU-001", "expected_qty": 10},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/purchase-orders/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPOController_List_Returns200(t *testing.T) {
	views := []responses.PurchaseOrderView{
		*samplePOCtrlView("po-1", "draft"),
		*samplePOCtrlView("po-2", "submitted"),
	}
	repo := &mockPOCtrlRepo{listed: views}
	r := newPOTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/purchase-orders/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPOController_GetByID_Returns200(t *testing.T) {
	view := samplePOCtrlView("po-1", "draft")
	repo := &mockPOCtrlRepo{byID: map[string]*responses.PurchaseOrderView{"po-1": view}}
	r := newPOTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/purchase-orders/po-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPOController_GetByID_Returns404_WhenNotFound(t *testing.T) {
	repo := &mockPOCtrlRepo{}
	r := newPOTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/purchase-orders/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPOController_Update_Returns200(t *testing.T) {
	repo := &mockPOCtrlRepo{updated: samplePOCtrlView("po-1", "draft")}
	r := newPOTestRouter(repo)

	body := map[string]interface{}{"notes": "updated"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/purchase-orders/po-1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPOController_Update_Returns400_WhenNotDraft(t *testing.T) {
	repoErr := &responses.InternalResponse{
		Message:    "Solo se pueden editar órdenes en estado 'draft'",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}
	repo := &mockPOCtrlRepo{updateErr: repoErr}
	r := newPOTestRouter(repo)

	body := map[string]interface{}{"notes": "x"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/purchase-orders/po-1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPOController_Delete_Returns200(t *testing.T) {
	repo := &mockPOCtrlRepo{}
	r := newPOTestRouter(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/purchase-orders/po-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ─────────────────────────────────────────────────────────────────────────────
// PO2 — Lifecycle controller tests
// ─────────────────────────────────────────────────────────────────────────────

func TestPOController_Submit_Returns200_WithReceivingTaskID(t *testing.T) {
	submitted := samplePOCtrlView("po-1", "submitted")
	rtID := "rt-001"
	repo := &mockPOCtrlRepo{submitView: submitted, submitRTID: rtID}
	r := newPOTestRouter(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/purchase-orders/po-1/submit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	// The response data should contain new_receiving_task_id.
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "expected data object in response")
	assert.Equal(t, rtID, data["new_receiving_task_id"])
}

func TestPOController_Submit_Returns400_WhenNotDraft(t *testing.T) {
	repoErr := &responses.InternalResponse{
		Message:    "Solo se pueden someter órdenes en estado 'draft'",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}
	repo := &mockPOCtrlRepo{submitErr: repoErr}
	r := newPOTestRouter(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/purchase-orders/po-1/submit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPOController_Cancel_Returns200(t *testing.T) {
	cancelled := samplePOCtrlView("po-1", "cancelled")
	repo := &mockPOCtrlRepo{cancelView: cancelled}
	r := newPOTestRouter(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/purchase-orders/po-1/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPOController_Cancel_Returns400_AlreadyCompleted(t *testing.T) {
	repoErr := &responses.InternalResponse{
		Message:    "No se puede cancelar una orden ya completada",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}
	repo := &mockPOCtrlRepo{cancelErr: repoErr}
	r := newPOTestRouter(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/purchase-orders/po-1/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
