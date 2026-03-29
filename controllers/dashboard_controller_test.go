package controllers

import (
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockDashboardRepoCtrl struct {
	dashboardStats    map[string]interface{}
	dashboardStatsErr *responses.InternalResponse
	inventorySummary  map[string]interface{}
	inventorySumErr   *responses.InternalResponse
	movementsMonthly  map[string]interface{}
	movementsErr      *responses.InternalResponse
	recentActivity    map[string]interface{}
	recentActivityErr *responses.InternalResponse
}

func (m *mockDashboardRepoCtrl) GetDashboardStats(tasksPeriod string, lowStockThreshold int) (map[string]interface{}, *responses.InternalResponse) {
	return m.dashboardStats, m.dashboardStatsErr
}

func (m *mockDashboardRepoCtrl) GetInventorySummary(period string) (map[string]interface{}, *responses.InternalResponse) {
	return m.inventorySummary, m.inventorySumErr
}

func (m *mockDashboardRepoCtrl) GetMovementsMonthly(period string) (map[string]interface{}, *responses.InternalResponse) {
	return m.movementsMonthly, m.movementsErr
}

func (m *mockDashboardRepoCtrl) GetRecentActivity() (map[string]interface{}, *responses.InternalResponse) {
	return m.recentActivity, m.recentActivityErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newDashboardController(repo *mockDashboardRepoCtrl) *DashboardController {
	svc := services.NewDashboardService(repo)
	return NewDashboardController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestDashboardController_GetDashboardStats_Success(t *testing.T) {
	repo := &mockDashboardRepoCtrl{
		dashboardStats: map[string]interface{}{
			"total_articles": 100,
			"pending_tasks":  5,
		},
	}
	ctrl := newDashboardController(repo)
	w := performRequest(ctrl.GetDashboardStats, "GET", "/dashboard/stats", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardController_GetDashboardStats_Empty(t *testing.T) {
	ctrl := newDashboardController(&mockDashboardRepoCtrl{dashboardStats: map[string]interface{}{}})
	w := performRequest(ctrl.GetDashboardStats, "GET", "/dashboard/stats", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardController_GetDashboardStats_ServiceError(t *testing.T) {
	repo := &mockDashboardRepoCtrl{
		dashboardStatsErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newDashboardController(repo)
	w := performRequest(ctrl.GetDashboardStats, "GET", "/dashboard/stats", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDashboardController_GetInventorySummary_Success(t *testing.T) {
	repo := &mockDashboardRepoCtrl{
		inventorySummary: map[string]interface{}{
			"total_skus":       50,
			"total_quantity":   1200,
			"low_stock_count":  3,
		},
	}
	ctrl := newDashboardController(repo)
	w := performRequest(ctrl.GetInventorySummary, "GET", "/dashboard/inventory-summary", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardController_GetInventorySummary_ServiceError(t *testing.T) {
	repo := &mockDashboardRepoCtrl{
		inventorySumErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newDashboardController(repo)
	w := performRequest(ctrl.GetInventorySummary, "GET", "/dashboard/inventory-summary", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDashboardController_GetMovementsMonthly_Success(t *testing.T) {
	repo := &mockDashboardRepoCtrl{
		movementsMonthly: map[string]interface{}{
			"months": []string{"Jan", "Feb", "Mar"},
			"inbound":  []int{10, 20, 15},
			"outbound": []int{5, 12, 8},
		},
	}
	ctrl := newDashboardController(repo)
	w := performRequest(ctrl.GetMovementsMonthly, "GET", "/dashboard/movements", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardController_GetMovementsMonthly_ServiceError(t *testing.T) {
	repo := &mockDashboardRepoCtrl{
		movementsErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newDashboardController(repo)
	w := performRequest(ctrl.GetMovementsMonthly, "GET", "/dashboard/movements", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDashboardController_GetRecentActivity_Success(t *testing.T) {
	repo := &mockDashboardRepoCtrl{
		recentActivity: map[string]interface{}{
			"activities": []string{"task completed", "item received"},
		},
	}
	ctrl := newDashboardController(repo)
	w := performRequest(ctrl.GetRecentActivity, "GET", "/dashboard/recent-activity", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardController_GetRecentActivity_ServiceError(t *testing.T) {
	repo := &mockDashboardRepoCtrl{
		recentActivityErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newDashboardController(repo)
	w := performRequest(ctrl.GetRecentActivity, "GET", "/dashboard/recent-activity", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
