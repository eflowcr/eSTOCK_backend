package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// mock repository
// ─────────────────────────────────────────────────────────────────────────────

type mockBackorderRepo struct {
	listResult    *responses.BackorderListResponse
	listErr       *responses.InternalResponse
	getResult     *responses.BackorderResponse
	getErr        *responses.InternalResponse
	fulfillResult *responses.FulfillBackorderResult
	fulfillErr    *responses.InternalResponse
}

func (m *mockBackorderRepo) List(_ string, _, _ *string, _, _ int) (*responses.BackorderListResponse, *responses.InternalResponse) {
	return m.listResult, m.listErr
}
func (m *mockBackorderRepo) GetByID(_, _ string) (*responses.BackorderResponse, *responses.InternalResponse) {
	return m.getResult, m.getErr
}
func (m *mockBackorderRepo) Fulfill(_, _, _ string) (*responses.FulfillBackorderResult, *responses.InternalResponse) {
	return m.fulfillResult, m.fulfillErr
}

// ─────────────────────────────────────────────────────────────────────────────
// tests
// ─────────────────────────────────────────────────────────────────────────────

func TestBackordersService_List_OK(t *testing.T) {
	expected := &responses.BackorderListResponse{Total: 3, Page: 1, Limit: 50}
	repo := &mockBackorderRepo{listResult: expected}
	svc := NewBackordersService(repo)

	result, resp := svc.List("tenant-1", nil, nil, 1, 50)
	require.Nil(t, resp)
	require.Equal(t, expected, result)
}

func TestBackordersService_List_WithFilters(t *testing.T) {
	status := "pending"
	expected := &responses.BackorderListResponse{Total: 1}
	repo := &mockBackorderRepo{listResult: expected}
	svc := NewBackordersService(repo)

	result, resp := svc.List("tenant-1", &status, nil, 1, 50)
	require.Nil(t, resp)
	require.Equal(t, expected, result)
}

func TestBackordersService_List_Error(t *testing.T) {
	repo := &mockBackorderRepo{listErr: &responses.InternalResponse{
		Error: errors.New("db error"), Message: "Error",
	}}
	svc := NewBackordersService(repo)

	result, resp := svc.List("tenant-1", nil, nil, 1, 50)
	require.Nil(t, result)
	require.NotNil(t, resp)
}

func TestBackordersService_GetByID_OK(t *testing.T) {
	expected := &responses.BackorderResponse{ID: "bo-1", Status: "pending", RemainingQty: 5}
	repo := &mockBackorderRepo{getResult: expected}
	svc := NewBackordersService(repo)

	bo, resp := svc.GetByID("bo-1", "tenant-1")
	require.Nil(t, resp)
	require.Equal(t, expected, bo)
}

func TestBackordersService_GetByID_NotFound(t *testing.T) {
	repo := &mockBackorderRepo{getErr: &responses.InternalResponse{
		Message:    "Backorder no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}}
	svc := NewBackordersService(repo)

	bo, resp := svc.GetByID("nonexistent", "tenant-1")
	require.Nil(t, bo)
	require.NotNil(t, resp)
	require.True(t, resp.Handled)
}

func TestBackordersService_Fulfill_OK(t *testing.T) {
	pickID := "pick-xyz"
	expected := &responses.FulfillBackorderResult{
		Backorder:     &responses.BackorderResponse{ID: "bo-1", Status: "pending"},
		PickingTaskID: pickID,
	}
	repo := &mockBackorderRepo{fulfillResult: expected}
	svc := NewBackordersService(repo)

	result, resp := svc.Fulfill("bo-1", "tenant-1", "user-1")
	require.Nil(t, resp)
	require.Equal(t, pickID, result.PickingTaskID)
}

func TestBackordersService_Fulfill_NotPending(t *testing.T) {
	repo := &mockBackorderRepo{fulfillErr: &responses.InternalResponse{
		Message:    "Solo se pueden fulfilliar backorders en estado 'pending'",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}}
	svc := NewBackordersService(repo)

	result, resp := svc.Fulfill("bo-1", "tenant-1", "user-1")
	require.Nil(t, result)
	require.NotNil(t, resp)
	require.True(t, resp.Handled)
}

func TestBackordersService_Fulfill_NoStock(t *testing.T) {
	repo := &mockBackorderRepo{fulfillErr: &responses.InternalResponse{
		Message:    "No hay stock disponible para fulfilliar este backorder",
		Handled:    true,
		StatusCode: responses.StatusBadRequest,
	}}
	svc := NewBackordersService(repo)

	result, resp := svc.Fulfill("bo-1", "tenant-1", "user-1")
	require.Nil(t, result)
	require.NotNil(t, resp)
}
