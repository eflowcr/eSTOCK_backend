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

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockStockAlertsRepoCtrl struct {
	alerts          []database.StockAlert
	alertsErr       *responses.InternalResponse
	analyzeResp     *responses.StockAlertResponse
	analyzeErr      *responses.InternalResponse
	lotExpResp      *responses.StockAlertResponse
	lotExpErr       *responses.InternalResponse
	resolveErr      *responses.InternalResponse
	exportData      []byte
	exportErr       *responses.InternalResponse
}

// S3.5 W2-B: tenantID is now part of every repo method; tests pass it through but
// the mock ignores it (we only verify the controller-side wiring works).
func (m *mockStockAlertsRepoCtrl) GetAllStockAlerts(_ string, resolved bool) ([]database.StockAlert, *responses.InternalResponse) {
	return m.alerts, m.alertsErr
}

func (m *mockStockAlertsRepoCtrl) Analyze(_ string) (*responses.StockAlertResponse, *responses.InternalResponse) {
	return m.analyzeResp, m.analyzeErr
}

func (m *mockStockAlertsRepoCtrl) LotExpiration(_ string) (*responses.StockAlertResponse, *responses.InternalResponse) {
	return m.lotExpResp, m.lotExpErr
}

func (m *mockStockAlertsRepoCtrl) ResolveAlert(_, alertID string) *responses.InternalResponse {
	return m.resolveErr
}

func (m *mockStockAlertsRepoCtrl) ExportAlertsToExcel(_ string) ([]byte, *responses.InternalResponse) {
	return m.exportData, m.exportErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newStockAlertsController(repo *mockStockAlertsRepoCtrl) *StockAlertsController {
	svc := services.NewStockAlertsService(repo)
	return NewStockAlertsController(*svc, ctrlTestTenantID)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestStockAlertsController_GetAllStockAlerts_Empty(t *testing.T) {
	ctrl := newStockAlertsController(&mockStockAlertsRepoCtrl{alerts: []database.StockAlert{}})
	w := performRequest(ctrl.GetAllStockAlerts, "GET", "/stock-alerts", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockAlertsController_GetAllStockAlerts_WithData(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		alerts: []database.StockAlert{
			{ID: "a-1", SKU: "SKU-001", AlertType: "low_stock", AlertLevel: "critical", Message: "Low stock"},
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.GetAllStockAlerts, "GET", "/stock-alerts", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockAlertsController_GetAllStockAlerts_ServiceError(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		alertsErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.GetAllStockAlerts, "GET", "/stock-alerts", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStockAlertsController_Analyze_Success(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		analyzeResp: &responses.StockAlertResponse{
			Message: "analysis complete",
			Alerts:  []database.StockAlert{{ID: "a-1", SKU: "SKU-001", AlertType: "low_stock"}},
			Summary: responses.StockAlertSumary{Total: 1, Critical: 1},
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.Analyze, "POST", "/stock-alerts/analyze", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockAlertsController_Analyze_ServiceError(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		analyzeErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.Analyze, "POST", "/stock-alerts/analyze", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStockAlertsController_LotExpiration_Success(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		lotExpResp: &responses.StockAlertResponse{
			Message: "expiration check complete",
			Alerts:  []database.StockAlert{{ID: "a-2", SKU: "SKU-002", AlertType: "expiring"}},
			Summary: responses.StockAlertSumary{Total: 1, Expiring: 1},
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.LotExpiration, "POST", "/stock-alerts/lot-expiration", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockAlertsController_LotExpiration_ServiceError(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		lotExpErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.LotExpiration, "POST", "/stock-alerts/lot-expiration", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStockAlertsController_ResolveAlert_Success(t *testing.T) {
	ctrl := newStockAlertsController(&mockStockAlertsRepoCtrl{})
	w := performRequest(ctrl.ResolveAlert, "PATCH", "/stock-alerts/a-1/resolve", nil, gin.Params{{Key: "id", Value: "a-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockAlertsController_ResolveAlert_MissingParam(t *testing.T) {
	ctrl := newStockAlertsController(&mockStockAlertsRepoCtrl{})
	w := performRequest(ctrl.ResolveAlert, "PATCH", "/stock-alerts//resolve", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockAlertsController_ResolveAlert_NotFound(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		resolveErr: &responses.InternalResponse{
			Message:    "alert not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.ResolveAlert, "PATCH", "/stock-alerts/a-99/resolve", nil, gin.Params{{Key: "id", Value: "a-99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStockAlertsController_ResolveAlert_ServiceError(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		resolveErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.ResolveAlert, "PATCH", "/stock-alerts/a-1/resolve", nil, gin.Params{{Key: "id", Value: "a-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStockAlertsController_ExportAlertsToExcel_Success(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{exportData: []byte("xlsx")}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.ExportAlertsToExcel, "GET", "/stock-alerts/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockAlertsController_ExportAlertsToExcel_NoData(t *testing.T) {
	ctrl := newStockAlertsController(&mockStockAlertsRepoCtrl{exportData: nil})
	w := performRequest(ctrl.ExportAlertsToExcel, "GET", "/stock-alerts/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockAlertsController_ExportAlertsToExcel_ServiceError(t *testing.T) {
	repo := &mockStockAlertsRepoCtrl{
		exportErr: &responses.InternalResponse{
			Message:    "export failed",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newStockAlertsController(repo)
	w := performRequest(ctrl.ExportAlertsToExcel, "GET", "/stock-alerts/export", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
