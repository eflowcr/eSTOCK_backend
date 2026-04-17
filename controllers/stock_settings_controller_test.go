package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockStockSettingsRepo struct {
	settings  *database.StockSetting
	upsertErr *responses.InternalResponse
}

func (m *mockStockSettingsRepo) GetOrCreate(_ string) (*database.StockSetting, *responses.InternalResponse) {
	if m.settings != nil {
		return m.settings, nil
	}
	defaults := &database.StockSetting{
		TenantID:              "00000000-0000-0000-0000-000000000001",
		ValuationMethod:       "avco",
		PickBatchBasedOn:      "fefo",
		ExpiryAlertDays:       30,
		AutoReserveStock:      true,
		AllowPartialReservation: true,
		PartialDeliveryPolicy: "immediate",
	}
	return defaults, nil
}

func (m *mockStockSettingsRepo) Upsert(_ string, data *requests.UpdateStockSettingsRequest) (*database.StockSetting, *responses.InternalResponse) {
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	s := &database.StockSetting{
		ValuationMethod:       data.ValuationMethod,
		PickBatchBasedOn:      data.PickBatchBasedOn,
		ExpiryAlertDays:       data.ExpiryAlertDays,
		PartialDeliveryPolicy: data.PartialDeliveryPolicy,
	}
	return s, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func newStockSettingsController(repo *mockStockSettingsRepo) *StockSettingsController {
	svc := services.NewStockSettingsService(repo)
	return NewStockSettingsController(*svc, "00000000-0000-0000-0000-000000000001")
}

func newStockSettingsRouter(ctrl *StockSettingsController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/settings/stock", ctrl.Get)
	r.PATCH("/settings/stock", ctrl.Update)
	return r
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestStockSettingsController_Get_DefaultsCreated(t *testing.T) {
	repo := &mockStockSettingsRepo{}
	ctrl := newStockSettingsController(repo)
	r := newStockSettingsRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/settings/stock", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "avco", data["valuation_method"])
}

func TestStockSettingsController_Update_HappyPath(t *testing.T) {
	repo := &mockStockSettingsRepo{}
	ctrl := newStockSettingsController(repo)
	r := newStockSettingsRouter(ctrl)

	body := requests.UpdateStockSettingsRequest{
		ValuationMethod:       "fifo",
		PickBatchBasedOn:      "fifo",
		PartialDeliveryPolicy: "when_all_ready",
		ExpiryAlertDays:       15,
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/settings/stock", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "fifo", data["valuation_method"])
}

func TestStockSettingsController_Update_InvalidValuationMethod(t *testing.T) {
	repo := &mockStockSettingsRepo{}
	ctrl := newStockSettingsController(repo)
	r := newStockSettingsRouter(ctrl)

	body := map[string]interface{}{
		"valuation_method":        "invalid",
		"pick_batch_based_on":     "fefo",
		"partial_delivery_policy": "immediate",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/settings/stock", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockSettingsController_Update_InvalidPickPolicy(t *testing.T) {
	repo := &mockStockSettingsRepo{}
	ctrl := newStockSettingsController(repo)
	r := newStockSettingsRouter(ctrl)

	body := map[string]interface{}{
		"valuation_method":        "avco",
		"pick_batch_based_on":     "invalid",
		"partial_delivery_policy": "immediate",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/settings/stock", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockSettingsController_GetOrCreate_Idempotent(t *testing.T) {
	fixed := &database.StockSetting{
		ValuationMethod:       "avco",
		PickBatchBasedOn:      "fefo",
		ExpiryAlertDays:       30,
		PartialDeliveryPolicy: "immediate",
	}
	repo := &mockStockSettingsRepo{settings: fixed}
	ctrl := newStockSettingsController(repo)
	r := newStockSettingsRouter(ctrl)

	// Call twice — same result both times
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/settings/stock", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
