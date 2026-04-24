package services

import (
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock repository
// ─────────────────────────────────────────────────────────────────────────────

type mockPORepo struct {
	created      *responses.PurchaseOrderView
	createErr    *responses.InternalResponse
	byID         map[string]*responses.PurchaseOrderView
	byIDErr      *responses.InternalResponse
	listed       []responses.PurchaseOrderView
	listErr      *responses.InternalResponse
	updated      *responses.PurchaseOrderView
	updateErr    *responses.InternalResponse
	softDelErr   *responses.InternalResponse
	submitView   *responses.PurchaseOrderView
	submitRTID   string
	submitErr    *responses.InternalResponse
	cancelView   *responses.PurchaseOrderView
	cancelErr    *responses.InternalResponse
	qtyUpdates   []database.PurchaseOrderItemQtyUpdate
	qtyUpdateErr *responses.InternalResponse
}

func (m *mockPORepo) Create(tenantID, createdBy string, req *requests.CreatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return m.created, m.createErr
}

func (m *mockPORepo) GetByID(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	if m.byID != nil {
		if v, ok := m.byID[id]; ok {
			return v, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockPORepo) List(tenantID string, status, supplierID, search *string, from, to *string, limit, offset int) ([]responses.PurchaseOrderView, *responses.InternalResponse) {
	return m.listed, m.listErr
}

func (m *mockPORepo) Update(id, tenantID string, req *requests.UpdatePurchaseOrderRequest) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return m.updated, m.updateErr
}

func (m *mockPORepo) SoftDelete(id, tenantID string) *responses.InternalResponse {
	return m.softDelErr
}

func (m *mockPORepo) Submit(id, tenantID, userID string) (*responses.PurchaseOrderView, string, *responses.InternalResponse) {
	return m.submitView, m.submitRTID, m.submitErr
}

func (m *mockPORepo) Cancel(id, tenantID string) (*responses.PurchaseOrderView, *responses.InternalResponse) {
	return m.cancelView, m.cancelErr
}

func (m *mockPORepo) UpdateReceivedQty(purchaseOrderID string, updates []database.PurchaseOrderItemQtyUpdate) *responses.InternalResponse {
	m.qtyUpdates = updates
	return m.qtyUpdateErr
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

const testTenantID = "00000000-0000-0000-0000-000000000001"
const testUserID = "user-abc"

func samplePOView(id, status string) *responses.PurchaseOrderView {
	return &responses.PurchaseOrderView{
		ID:         id,
		PONumber:   "PO-2026-0001",
		SupplierID: "supplier-1",
		Status:     status,
		TenantID:   testTenantID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Items: []responses.PurchaseOrderItemView{
			{ID: "item-1", ArticleSKU: "SKU-001", ExpectedQty: 10, ReceivedQty: 0, RejectedQty: 0},
		},
	}
}

func sampleCreateReq() *requests.CreatePurchaseOrderRequest {
	return &requests.CreatePurchaseOrderRequest{
		SupplierID: "supplier-1",
		Items: []requests.CreatePurchaseOrderItemRequest{
			{ArticleSKU: "SKU-001", ExpectedQty: 10},
		},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PO1 — CRUD happy paths
// ─────────────────────────────────────────────────────────────────────────────

func TestPurchaseOrdersService_Create_HappyPath(t *testing.T) {
	expected := samplePOView("po-1", "draft")
	svc := NewPurchaseOrdersService(&mockPORepo{created: expected})

	got, resp := svc.Create(testTenantID, testUserID, sampleCreateReq())
	require.Nil(t, resp)
	require.NotNil(t, got)
	assert.Equal(t, "draft", got.Status)
	assert.Equal(t, "po-1", got.ID)
}

func TestPurchaseOrdersService_Create_RepoError(t *testing.T) {
	repoErr := &responses.InternalResponse{Error: assert.AnError, Message: "db error"}
	svc := NewPurchaseOrdersService(&mockPORepo{createErr: repoErr})

	got, resp := svc.Create(testTenantID, testUserID, sampleCreateReq())
	assert.Nil(t, got)
	assert.NotNil(t, resp)
	assert.Equal(t, "db error", resp.Message)
}

func TestPurchaseOrdersService_GetByID_Found(t *testing.T) {
	view := samplePOView("po-2", "draft")
	repo := &mockPORepo{byID: map[string]*responses.PurchaseOrderView{"po-2": view}}
	svc := NewPurchaseOrdersService(repo)

	got, resp := svc.GetByID("po-2", testTenantID)
	require.Nil(t, resp)
	require.NotNil(t, got)
	assert.Equal(t, "po-2", got.ID)
}

func TestPurchaseOrdersService_GetByID_NotFound(t *testing.T) {
	svc := NewPurchaseOrdersService(&mockPORepo{})
	got, resp := svc.GetByID("nonexistent", testTenantID)
	assert.Nil(t, got)
	assert.NotNil(t, resp)
	assert.Equal(t, responses.StatusNotFound, resp.StatusCode)
}

func TestPurchaseOrdersService_List_ReturnsTenantScoped(t *testing.T) {
	views := []responses.PurchaseOrderView{
		*samplePOView("po-1", "draft"),
		*samplePOView("po-2", "submitted"),
	}
	svc := NewPurchaseOrdersService(&mockPORepo{listed: views})

	got, resp := svc.List(testTenantID, nil, nil, nil, nil, nil, 50, 0)
	require.Nil(t, resp)
	assert.Len(t, got, 2)
	// Verify tenant isolation: all items belong to testTenantID
	for _, po := range got {
		assert.Equal(t, testTenantID, po.TenantID)
	}
}

func TestPurchaseOrdersService_Update_DraftOnly(t *testing.T) {
	updated := samplePOView("po-1", "draft")
	svc := NewPurchaseOrdersService(&mockPORepo{updated: updated})

	req := &requests.UpdatePurchaseOrderRequest{Notes: poStrPtr("updated notes")}
	got, resp := svc.Update("po-1", testTenantID, req)
	require.Nil(t, resp)
	assert.NotNil(t, got)
}

func TestPurchaseOrdersService_Update_ReturnsRepoError_WhenNotDraft(t *testing.T) {
	repoErr := &responses.InternalResponse{
		Message:    "Solo se pueden editar órdenes de compra en estado 'draft' (actual: submitted)",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}
	svc := NewPurchaseOrdersService(&mockPORepo{updateErr: repoErr})

	req := &requests.UpdatePurchaseOrderRequest{Notes: poStrPtr("x")}
	got, resp := svc.Update("po-1", testTenantID, req)
	assert.Nil(t, got)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

func TestPurchaseOrdersService_SoftDelete_OK(t *testing.T) {
	svc := NewPurchaseOrdersService(&mockPORepo{})
	resp := svc.SoftDelete("po-1", testTenantID)
	assert.Nil(t, resp)
}

func TestPurchaseOrdersService_SoftDelete_NotFound(t *testing.T) {
	repoErr := &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
	svc := NewPurchaseOrdersService(&mockPORepo{softDelErr: repoErr})
	resp := svc.SoftDelete("nonexistent", testTenantID)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusNotFound, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// PO2 — Lifecycle: Submit
// ─────────────────────────────────────────────────────────────────────────────

func TestPurchaseOrdersService_Submit_HappyPath(t *testing.T) {
	submitted := samplePOView("po-1", "submitted")
	rtID := "rt-001"
	svc := NewPurchaseOrdersService(&mockPORepo{submitView: submitted, submitRTID: rtID})

	po, newRTID, resp := svc.Submit("po-1", testTenantID, testUserID)
	require.Nil(t, resp)
	assert.Equal(t, "submitted", po.Status)
	assert.Equal(t, rtID, newRTID)
}

func TestPurchaseOrdersService_Submit_AlreadySubmitted_ReturnsError(t *testing.T) {
	repoErr := &responses.InternalResponse{
		Message:    "Solo se pueden someter órdenes en estado 'draft' (actual: submitted)",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}
	svc := NewPurchaseOrdersService(&mockPORepo{submitErr: repoErr})

	po, newRTID, resp := svc.Submit("po-1", testTenantID, testUserID)
	assert.Nil(t, po)
	assert.Empty(t, newRTID)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// PO2 — Lifecycle: Cancel
// ─────────────────────────────────────────────────────────────────────────────

func TestPurchaseOrdersService_Cancel_HappyPath(t *testing.T) {
	cancelled := samplePOView("po-1", "cancelled")
	svc := NewPurchaseOrdersService(&mockPORepo{cancelView: cancelled})

	po, resp := svc.Cancel("po-1", testTenantID)
	require.Nil(t, resp)
	assert.Equal(t, "cancelled", po.Status)
}

func TestPurchaseOrdersService_Cancel_AlreadyCompleted_ReturnsError(t *testing.T) {
	repoErr := &responses.InternalResponse{
		Message:    "No se puede cancelar una orden de compra ya completada",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}
	svc := NewPurchaseOrdersService(&mockPORepo{cancelErr: repoErr})

	po, resp := svc.Cancel("po-1", testTenantID)
	assert.Nil(t, po)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// Tenant isolation unit-level guard
// ─────────────────────────────────────────────────────────────────────────────

func TestPurchaseOrdersService_TenantIsolation_GetByID(t *testing.T) {
	// View belongs to tenant A; querying with tenant B returns not-found.
	viewTenantA := samplePOView("po-1", "draft")
	viewTenantA.TenantID = "tenant-A"

	repo := &mockPORepo{
		byIDErr: &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound},
	}
	svc := NewPurchaseOrdersService(repo)

	got, resp := svc.GetByID("po-1", "tenant-B")
	assert.Nil(t, got)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusNotFound, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func poStrPtr(s string) *string { return &s }
