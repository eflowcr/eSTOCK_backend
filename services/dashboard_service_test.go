package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDashboardRepo is an in-memory fake for unit testing DashboardService.
type mockDashboardRepo struct {
	dashboardStats    map[string]interface{}
	dashboardStatsErr *responses.InternalResponse
	inventorySummary    map[string]interface{}
	inventorySummaryErr *responses.InternalResponse
	movementsMonthly    map[string]interface{}
	movementsMonthlyErr *responses.InternalResponse
	recentActivity    map[string]interface{}
	recentActivityErr *responses.InternalResponse
}

func (m *mockDashboardRepo) GetDashboardStats(tasksPeriod string, lowStockThreshold int) (map[string]interface{}, *responses.InternalResponse) {
	return m.dashboardStats, m.dashboardStatsErr
}

func (m *mockDashboardRepo) GetInventorySummary(period string) (map[string]interface{}, *responses.InternalResponse) {
	return m.inventorySummary, m.inventorySummaryErr
}

func (m *mockDashboardRepo) GetMovementsMonthly(period string) (map[string]interface{}, *responses.InternalResponse) {
	return m.movementsMonthly, m.movementsMonthlyErr
}

func (m *mockDashboardRepo) GetRecentActivity() (map[string]interface{}, *responses.InternalResponse) {
	return m.recentActivity, m.recentActivityErr
}

func TestDashboardService_GetDashboardStats_Success(t *testing.T) {
	stats := map[string]interface{}{
		"total_articles":   42,
		"low_stock_count":  5,
		"tasks_completed":  10,
	}
	repo := &mockDashboardRepo{dashboardStats: stats}
	svc := NewDashboardService(repo)

	result, errResp := svc.GetDashboardStats("monthly", 10)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, 42, result["total_articles"])
	assert.Equal(t, 5, result["low_stock_count"])
}

func TestDashboardService_GetDashboardStats_Error(t *testing.T) {
	repo := &mockDashboardRepo{
		dashboardStatsErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching dashboard stats",
			Handled: false,
		},
	}
	svc := NewDashboardService(repo)

	result, errResp := svc.GetDashboardStats("weekly", 5)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestDashboardService_GetInventorySummary_Success(t *testing.T) {
	summary := map[string]interface{}{
		"total_value":   15000.50,
		"total_items":   200,
		"period":        "monthly",
	}
	repo := &mockDashboardRepo{inventorySummary: summary}
	svc := NewDashboardService(repo)

	result, errResp := svc.GetInventorySummary("monthly")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, 200, result["total_items"])
	assert.Equal(t, "monthly", result["period"])
}

func TestDashboardService_GetInventorySummary_Error(t *testing.T) {
	repo := &mockDashboardRepo{
		inventorySummaryErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching inventory summary",
			Handled: false,
		},
	}
	svc := NewDashboardService(repo)

	result, errResp := svc.GetInventorySummary("weekly")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestDashboardService_GetMovementsMonthly_Success(t *testing.T) {
	movements := map[string]interface{}{
		"inbound":  120,
		"outbound": 85,
		"period":   "monthly",
	}
	repo := &mockDashboardRepo{movementsMonthly: movements}
	svc := NewDashboardService(repo)

	result, errResp := svc.GetMovementsMonthly("monthly")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, 120, result["inbound"])
	assert.Equal(t, 85, result["outbound"])
}

func TestDashboardService_GetMovementsMonthly_Error(t *testing.T) {
	repo := &mockDashboardRepo{
		movementsMonthlyErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching monthly movements",
			Handled: false,
		},
	}
	svc := NewDashboardService(repo)

	result, errResp := svc.GetMovementsMonthly("yearly")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestDashboardService_GetRecentActivity_Success(t *testing.T) {
	activity := map[string]interface{}{
		"last_receiving": "2026-03-28",
		"last_picking":   "2026-03-29",
		"events_count":   15,
	}
	repo := &mockDashboardRepo{recentActivity: activity}
	svc := NewDashboardService(repo)

	result, errResp := svc.GetRecentActivity()
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, 15, result["events_count"])
	assert.Equal(t, "2026-03-29", result["last_picking"])
}

func TestDashboardService_GetRecentActivity_Error(t *testing.T) {
	repo := &mockDashboardRepo{
		recentActivityErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching recent activity",
			Handled: false,
		},
	}
	svc := NewDashboardService(repo)

	result, errResp := svc.GetRecentActivity()
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}
