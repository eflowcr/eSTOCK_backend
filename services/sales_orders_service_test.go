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
// Mock repo
// ─────────────────────────────────────────────────────────────────────────────

type mockSalesOrdersRepo struct {
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

func (m *mockSalesOrdersRepo) Create(_ string, _ string, _ *requests.CreateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	return m.createResult, m.createErr
}
func (m *mockSalesOrdersRepo) List(_ string, _, _, _, _, _ *string, _, _ int) (*responses.SalesOrderListResponse, *responses.InternalResponse) {
	return m.listResult, m.listErr
}
func (m *mockSalesOrdersRepo) GetByID(_ string, _ string) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	return m.getResult, m.getErr
}
func (m *mockSalesOrdersRepo) Update(_ string, _ string, _ *requests.UpdateSalesOrderRequest) (*responses.SalesOrderResponse, *responses.InternalResponse) {
	return m.updateResult, m.updateErr
}
func (m *mockSalesOrdersRepo) SoftDelete(_ string, _ string) *responses.InternalResponse {
	return m.deleteErr
}
func (m *mockSalesOrdersRepo) Submit(_ string, _ string, _ string) (*responses.SubmitSalesOrderResult, *responses.InternalResponse) {
	return m.submitResult, m.submitErr
}
func (m *mockSalesOrdersRepo) Cancel(_ string, _ string, _ string) *responses.InternalResponse {
	return m.cancelErr
}
func (m *mockSalesOrdersRepo) UpdatePickedQty(_ string, _ map[string]float64) (string, *responses.InternalResponse) {
	return "", m.updatePickErr
}

// mockClientRepo for customer validation.
type mockSOClientRepo struct {
	client *database.Client
	err    *responses.InternalResponse
}

func (m *mockSOClientRepo) GetByID(_ string) (*database.Client, *responses.InternalResponse) {
	return m.client, m.err
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

func validSOCreateReq() *requests.CreateSalesOrderRequest {
	return &requests.CreateSalesOrderRequest{
		CustomerID: "cust-001",
		Items: []requests.CreateSalesOrderItem{
			{ArticleSKU: "SKU-A", ExpectedQty: 10, UnitPrice: floatPtr(5.00)},
			{ArticleSKU: "SKU-B", ExpectedQty: 3},
		},
	}
}

func floatPtr(f float64) *float64 { return &f }

func sampleSOResponse(id, soNumber string) *responses.SalesOrderResponse {
	now := time.Now()
	return &responses.SalesOrderResponse{
		ID:         id,
		TenantID:   "tenant-1",
		SONumber:   soNumber,
		CustomerID: "cust-001",
		Status:     "draft",
		CreatedAt:  now,
		UpdatedAt:  now,
		Items:      []database.SalesOrderItem{},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SO1 — CRUD tests
// ─────────────────────────────────────────────────────────────────────────────

func TestSalesOrdersService_Create_OK(t *testing.T) {
	repo := &mockSalesOrdersRepo{
		createResult: sampleSOResponse("so-1", "SO-2026-0001"),
	}
	svc := NewSalesOrdersService(repo)

	result, resp := svc.Create("tenant-1", "user-1", validSOCreateReq())
	require.Nil(t, resp)
	require.NotNil(t, result)
	assert.Equal(t, "SO-2026-0001", result.SONumber)
	assert.Equal(t, "draft", result.Status)
}

func TestSalesOrdersService_Create_DuplicateSKU(t *testing.T) {
	repo := &mockSalesOrdersRepo{}
	svc := NewSalesOrdersService(repo)

	req := &requests.CreateSalesOrderRequest{
		CustomerID: "cust-001",
		Items: []requests.CreateSalesOrderItem{
			{ArticleSKU: "SKU-A", ExpectedQty: 5},
			{ArticleSKU: "SKU-A", ExpectedQty: 3}, // duplicate
		},
	}
	result, resp := svc.Create("tenant-1", "user-1", req)
	require.NotNil(t, resp)
	require.Nil(t, result)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, resp.Message, "SKU duplicado")
}

func TestSalesOrdersService_Create_CustomerValidation_WrongType(t *testing.T) {
	repo := &mockSalesOrdersRepo{}
	svc := NewSalesOrdersService(repo)
	svc.ClientsService = &mockSOClientRepo{
		client: &database.Client{ID: "cust-001", Type: "supplier"},
	}

	result, resp := svc.Create("tenant-1", "user-1", validSOCreateReq())
	require.NotNil(t, resp)
	require.Nil(t, result)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, resp.Message, "tipo")
}

func TestSalesOrdersService_Create_CustomerValidation_OK(t *testing.T) {
	repo := &mockSalesOrdersRepo{
		createResult: sampleSOResponse("so-1", "SO-2026-0001"),
	}
	svc := NewSalesOrdersService(repo)
	svc.ClientsService = &mockSOClientRepo{
		client: &database.Client{ID: "cust-001", Type: "customer"},
	}

	result, resp := svc.Create("tenant-1", "user-1", validSOCreateReq())
	require.Nil(t, resp)
	require.NotNil(t, result)
}

func TestSalesOrdersService_Create_RepoError(t *testing.T) {
	repo := &mockSalesOrdersRepo{
		createErr: &responses.InternalResponse{Message: "DB error", StatusCode: responses.StatusInternalServerError},
	}
	svc := NewSalesOrdersService(repo)

	result, resp := svc.Create("tenant-1", "user-1", validSOCreateReq())
	require.NotNil(t, resp)
	require.Nil(t, result)
}

func TestSalesOrdersService_GetByID_OK(t *testing.T) {
	expected := sampleSOResponse("so-1", "SO-2026-0001")
	repo := &mockSalesOrdersRepo{getResult: expected}
	svc := NewSalesOrdersService(repo)

	result, resp := svc.GetByID("so-1", "tenant-1")
	require.Nil(t, resp)
	assert.Equal(t, "so-1", result.ID)
}

func TestSalesOrdersService_GetByID_NotFound(t *testing.T) {
	repo := &mockSalesOrdersRepo{
		getErr: &responses.InternalResponse{Message: "not found", StatusCode: responses.StatusNotFound},
	}
	svc := NewSalesOrdersService(repo)

	result, resp := svc.GetByID("nonexistent", "tenant-1")
	require.NotNil(t, resp)
	require.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, resp.StatusCode)
}

func TestSalesOrdersService_List_OK(t *testing.T) {
	expected := &responses.SalesOrderListResponse{
		Items:      []responses.SalesOrderListItem{{ID: "so-1", SONumber: "SO-2026-0001"}},
		Total:      1,
		Page:       1,
		Limit:      20,
		TotalPages: 1,
	}
	repo := &mockSalesOrdersRepo{listResult: expected}
	svc := NewSalesOrdersService(repo)

	result, resp := svc.List("tenant-1", nil, nil, nil, nil, nil, 1, 20)
	require.Nil(t, resp)
	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total)
}

func TestSalesOrdersService_Update_OK(t *testing.T) {
	updated := sampleSOResponse("so-1", "SO-2026-0001")
	repo := &mockSalesOrdersRepo{updateResult: updated}
	svc := NewSalesOrdersService(repo)

	req := &requests.UpdateSalesOrderRequest{Notes: soStrPtr("updated note")}
	result, resp := svc.Update("so-1", "tenant-1", req)
	require.Nil(t, resp)
	assert.NotNil(t, result)
}

func TestSalesOrdersService_SoftDelete_OK(t *testing.T) {
	repo := &mockSalesOrdersRepo{}
	svc := NewSalesOrdersService(repo)

	resp := svc.SoftDelete("so-1", "tenant-1")
	require.Nil(t, resp)
}

func TestSalesOrdersService_SoftDelete_Error(t *testing.T) {
	repo := &mockSalesOrdersRepo{
		deleteErr: &responses.InternalResponse{Message: "Solo drafts", StatusCode: responses.StatusBadRequest},
	}
	svc := NewSalesOrdersService(repo)

	resp := svc.SoftDelete("so-1", "tenant-1")
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO2 — Lifecycle tests
// ─────────────────────────────────────────────────────────────────────────────

func TestSalesOrdersService_Submit_OK(t *testing.T) {
	ptID := "pt-abc"
	expected := &responses.SubmitSalesOrderResult{
		SalesOrder:    sampleSOResponse("so-1", "SO-2026-0001"),
		PickingTaskID: ptID,
	}
	expected.SalesOrder.Status = "submitted"
	expected.SalesOrder.PickingTaskID = &ptID

	repo := &mockSalesOrdersRepo{submitResult: expected}
	svc := NewSalesOrdersService(repo)

	result, resp := svc.Submit("so-1", "tenant-1", "user-1")
	require.Nil(t, resp)
	require.NotNil(t, result)
	assert.Equal(t, ptID, result.PickingTaskID)
	assert.Equal(t, "submitted", result.SalesOrder.Status)
}

func TestSalesOrdersService_Submit_NotDraft(t *testing.T) {
	repo := &mockSalesOrdersRepo{
		submitErr: &responses.InternalResponse{Message: "Solo drafts", StatusCode: responses.StatusBadRequest},
	}
	svc := NewSalesOrdersService(repo)

	result, resp := svc.Submit("so-1", "tenant-1", "user-1")
	require.NotNil(t, resp)
	require.Nil(t, result)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

func TestSalesOrdersService_Cancel_OK(t *testing.T) {
	repo := &mockSalesOrdersRepo{}
	svc := NewSalesOrdersService(repo)

	resp := svc.Cancel("so-1", "tenant-1", "user-1")
	require.Nil(t, resp)
}

func TestSalesOrdersService_Cancel_Completed(t *testing.T) {
	repo := &mockSalesOrdersRepo{
		cancelErr: &responses.InternalResponse{Message: "ya completada", StatusCode: responses.StatusBadRequest},
	}
	svc := NewSalesOrdersService(repo)

	resp := svc.Cancel("so-1", "tenant-1", "user-1")
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// SO3 — UpdatePickedQty
// ─────────────────────────────────────────────────────────────────────────────

func TestSalesOrdersService_UpdatePickedQty_OK(t *testing.T) {
	repo := &mockSalesOrdersRepo{}
	svc := NewSalesOrdersService(repo)

	_, resp := svc.UpdatePickedQty("so-1", map[string]float64{"SKU-A": 10, "SKU-B": 3})
	require.Nil(t, resp)
}

// ─────────────────────────────────────────────────────────────────────────────
// Tenant isolation: List scoped by tenantID
// ─────────────────────────────────────────────────────────────────────────────

func TestSalesOrdersService_TenantIsolation_List(t *testing.T) {
	// tenant-A has 1 order, tenant-B has 0.
	repoA := &mockSalesOrdersRepo{
		listResult: &responses.SalesOrderListResponse{
			Items: []responses.SalesOrderListItem{{ID: "so-a1", SONumber: "SO-2026-0001"}},
			Total: 1, Page: 1, Limit: 20, TotalPages: 1,
		},
	}
	repoB := &mockSalesOrdersRepo{
		listResult: &responses.SalesOrderListResponse{
			Items: []responses.SalesOrderListItem{},
			Total: 0, Page: 1, Limit: 20, TotalPages: 0,
		},
	}

	svcA := NewSalesOrdersService(repoA)
	svcB := NewSalesOrdersService(repoB)

	resA, _ := svcA.List("tenant-A", nil, nil, nil, nil, nil, 1, 20)
	resB, _ := svcB.List("tenant-B", nil, nil, nil, nil, nil, 1, 20)

	assert.Equal(t, int64(1), resA.Total)
	assert.Equal(t, int64(0), resB.Total)
	// Ensure tenant-B cannot see tenant-A's orders.
	for _, item := range resB.Items {
		assert.NotEqual(t, "so-a1", item.ID)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers for test
// ─────────────────────────────────────────────────────────────────────────────

func soStrPtr(s string) *string { return &s }
