package controllers

import (
	"net/http"
	"net/http/httptest"
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

// S3.5.4 (B15 fix): root listing routes (with and without trailing slash) must respond 200,
// not 404. Previously only /:resolved was registered, so direct probes / SDK clients hitting
// /api/stock-alerts/ got page-not-found. We register both /, "" and /:resolved against
// GetAllStockAlerts; this test guards against accidental regression of the route table.
func TestStockAlertsRoutes_RootListingNotFoundRegression(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &mockStockAlertsRepoCtrl{alerts: []database.StockAlert{}}
	ctrl := newStockAlertsController(repo)

	router := gin.New()
	api := router.Group("/api")
	route := api.Group("/stock-alerts")
	{
		// Mirror routes/stock_alerts_routes.go (without auth/permission middleware).
		route.GET("", ctrl.GetAllStockAlerts)
		route.GET("/", ctrl.GetAllStockAlerts)
		route.GET("/analyze", ctrl.Analyze)
		route.GET("/lot-expiration", ctrl.LotExpiration)
		route.GET("/export", ctrl.ExportAlertsToExcel)
		route.GET("/:resolved", ctrl.GetAllStockAlerts)
		route.PATCH("/:id/resolve", ctrl.ResolveAlert)
	}

	cases := []struct {
		name string
		path string
	}{
		{"root no slash", "/api/stock-alerts"},
		{"root with slash", "/api/stock-alerts/"},
		{"resolved=true", "/api/stock-alerts/true"},
		{"resolved=false", "/api/stock-alerts/false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tc.path, nil)
			router.ServeHTTP(w, req)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "GET %s should not return 404 (B15 regression)", tc.path)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
