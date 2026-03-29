package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStockAlertsRepo is an in-memory fake for unit testing StockAlertsService.
type mockStockAlertsRepo struct {
	alerts           []database.StockAlert
	alertsErr        *responses.InternalResponse
	analyzeResp      *responses.StockAlertResponse
	analyzeErr       *responses.InternalResponse
	lotExpirationResp *responses.StockAlertResponse
	lotExpirationErr  *responses.InternalResponse
	resolveErr       *responses.InternalResponse
	excelBytes       []byte
	excelErr         *responses.InternalResponse
}

func (m *mockStockAlertsRepo) GetAllStockAlerts(resolved bool) ([]database.StockAlert, *responses.InternalResponse) {
	return m.alerts, m.alertsErr
}

func (m *mockStockAlertsRepo) Analyze() (*responses.StockAlertResponse, *responses.InternalResponse) {
	return m.analyzeResp, m.analyzeErr
}

func (m *mockStockAlertsRepo) LotExpiration() (*responses.StockAlertResponse, *responses.InternalResponse) {
	return m.lotExpirationResp, m.lotExpirationErr
}

func (m *mockStockAlertsRepo) ResolveAlert(alertID string) *responses.InternalResponse {
	return m.resolveErr
}

func (m *mockStockAlertsRepo) ExportAlertsToExcel() ([]byte, *responses.InternalResponse) {
	return m.excelBytes, m.excelErr
}

func TestStockAlertsService_GetAllStockAlerts_Success(t *testing.T) {
	alerts := []database.StockAlert{
		{ID: "alert-1", SKU: "SKU1", AlertType: "low_stock", AlertLevel: "critical"},
		{ID: "alert-2", SKU: "SKU2", AlertType: "low_stock", AlertLevel: "high"},
	}
	repo := &mockStockAlertsRepo{alerts: alerts}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.GetAllStockAlerts(false)
	require.Nil(t, errResp)
	require.Len(t, result, 2)
	assert.Equal(t, "SKU1", result[0].SKU)
	assert.Equal(t, "critical", result[0].AlertLevel)
}

func TestStockAlertsService_GetAllStockAlerts_Resolved(t *testing.T) {
	alerts := []database.StockAlert{
		{ID: "alert-3", SKU: "SKU3", IsResolved: true},
	}
	repo := &mockStockAlertsRepo{alerts: alerts}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.GetAllStockAlerts(true)
	require.Nil(t, errResp)
	require.Len(t, result, 1)
	assert.True(t, result[0].IsResolved)
}

func TestStockAlertsService_GetAllStockAlerts_Error(t *testing.T) {
	repo := &mockStockAlertsRepo{
		alertsErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching alerts",
			Handled: false,
		},
	}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.GetAllStockAlerts(false)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestStockAlertsService_Analyze_Success(t *testing.T) {
	analyzeResp := &responses.StockAlertResponse{
		Message: "Analysis complete",
		Alerts:  []database.StockAlert{{ID: "alert-1", SKU: "SKU1"}},
		Summary: responses.StockAlertSumary{Total: 1, Critical: 1},
	}
	repo := &mockStockAlertsRepo{analyzeResp: analyzeResp}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.Analyze()
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "Analysis complete", result.Message)
	assert.Equal(t, 1, result.Summary.Total)
	assert.Equal(t, 1, result.Summary.Critical)
}

func TestStockAlertsService_Analyze_Error(t *testing.T) {
	repo := &mockStockAlertsRepo{
		analyzeErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error analyzing stock",
			Handled: false,
		},
	}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.Analyze()
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestStockAlertsService_LotExpiration_Success(t *testing.T) {
	lotResp := &responses.StockAlertResponse{
		Message: "Lot expiration analysis complete",
		Alerts:  []database.StockAlert{{ID: "alert-4", SKU: "SKU4", AlertType: "expiration"}},
		Summary: responses.StockAlertSumary{Total: 1, Expiring: 1},
	}
	repo := &mockStockAlertsRepo{lotExpirationResp: lotResp}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.LotExpiration()
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "Lot expiration analysis complete", result.Message)
	assert.Equal(t, 1, result.Summary.Expiring)
}

func TestStockAlertsService_LotExpiration_Error(t *testing.T) {
	repo := &mockStockAlertsRepo{
		lotExpirationErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error checking lot expiration",
			Handled: false,
		},
	}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.LotExpiration()
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}

func TestStockAlertsService_ResolveAlert_Success(t *testing.T) {
	repo := &mockStockAlertsRepo{}
	svc := NewStockAlertsService(repo)

	errResp := svc.ResolveAlert("alert-1")
	require.Nil(t, errResp)
}

func TestStockAlertsService_ResolveAlert_NotFound(t *testing.T) {
	repo := &mockStockAlertsRepo{
		resolveErr: &responses.InternalResponse{
			Message:    "Alert not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewStockAlertsService(repo)

	errResp := svc.ResolveAlert("alert-99")
	require.NotNil(t, errResp)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestStockAlertsService_ExportAlertsToExcel_Success(t *testing.T) {
	excelBytes := []byte("excel-content")
	repo := &mockStockAlertsRepo{excelBytes: excelBytes}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.ExportAlertsToExcel()
	require.Nil(t, errResp)
	assert.Equal(t, excelBytes, result)
}

func TestStockAlertsService_ExportAlertsToExcel_Error(t *testing.T) {
	repo := &mockStockAlertsRepo{
		excelErr: &responses.InternalResponse{
			Error:   errors.New("export error"),
			Message: "Error exporting to Excel",
			Handled: false,
		},
	}
	svc := NewStockAlertsService(repo)

	result, errResp := svc.ExportAlertsToExcel()
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.False(t, errResp.Handled)
}
