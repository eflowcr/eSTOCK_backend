package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ───────────────────────────────────────────────────────────────

type mockInventoryRepoCtrl struct {
	inventory    []*dto.EnhancedInventory
	bySkuLoc     map[string]*dto.EnhancedInventory
	createErr    *responses.InternalResponse
	updateErr    *responses.InternalResponse
	deleteErr    *responses.InternalResponse
	lots         []responses.InventoryLot
	lotsErr      *responses.InternalResponse
	serials      []responses.InventorySerialWithSerial
	serialsErr   *responses.InternalResponse
	createLotErr *responses.InternalResponse
	deleteLotErr *responses.InternalResponse
	createSerErr *responses.InternalResponse
	deleteSerErr *responses.InternalResponse
	trend        *dto.ConsumptionTrend
	trendErr     *responses.InternalResponse
	suggestions  []dto.PickSuggestion
	suggestErr   *responses.InternalResponse
}

func (m *mockInventoryRepoCtrl) GetAllInventory() ([]*dto.EnhancedInventory, *responses.InternalResponse) {
	return m.inventory, nil
}

func (m *mockInventoryRepoCtrl) GetInventoryBySkuAndLocation(sku, location string) (*dto.EnhancedInventory, *responses.InternalResponse) {
	key := sku + ":" + location
	if m.bySkuLoc != nil {
		if item, ok := m.bySkuLoc[key]; ok {
			return item, nil
		}
	}
	return nil, nil
}

func (m *mockInventoryRepoCtrl) CreateInventory(userId string, item *requests.CreateInventory) *responses.InternalResponse {
	return m.createErr
}

func (m *mockInventoryRepoCtrl) UpdateInventory(item *requests.UpdateInventory) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockInventoryRepoCtrl) DeleteInventory(sku, location string) *responses.InternalResponse {
	return m.deleteErr
}

func (m *mockInventoryRepoCtrl) Trend(sku string) (*dto.ConsumptionTrend, *responses.InternalResponse) {
	return m.trend, m.trendErr
}

func (m *mockInventoryRepoCtrl) ImportInventoryFromExcel(userId string, fileBytes []byte) ([]string, []string, *responses.InternalResponse) {
	return []string{"inv-1"}, nil, nil
}

func (m *mockInventoryRepoCtrl) ImportInventoryFromJSON(userId string, rows []requests.InventoryImportRow) ([]string, []string, *responses.InternalResponse) {
	imported := make([]string, len(rows))
	for i, r := range rows {
		imported[i] = r.SKU
	}
	return imported, nil, nil
}

func (m *mockInventoryRepoCtrl) ValidateImportRows(rows []requests.InventoryImportRow) ([]responses.InventoryValidationResult, *responses.InternalResponse) {
	return []responses.InventoryValidationResult{}, nil
}

func (m *mockInventoryRepoCtrl) ExportInventoryToExcel() ([]byte, *responses.InternalResponse) {
	return []byte("xlsx"), nil
}

func (m *mockInventoryRepoCtrl) GetInventoryLots(inventoryID string) ([]responses.InventoryLot, *responses.InternalResponse) {
	return m.lots, m.lotsErr
}

func (m *mockInventoryRepoCtrl) GetInventorySerials(inventoryID string) ([]responses.InventorySerialWithSerial, *responses.InternalResponse) {
	return m.serials, m.serialsErr
}

func (m *mockInventoryRepoCtrl) CreateInventoryLot(id string, input *requests.CreateInventoryLotRequest) *responses.InternalResponse {
	return m.createLotErr
}

func (m *mockInventoryRepoCtrl) DeleteInventoryLot(id string) *responses.InternalResponse {
	return m.deleteLotErr
}

func (m *mockInventoryRepoCtrl) CreateInventorySerial(id string, input *requests.CreateInventorySerial) *responses.InternalResponse {
	return m.createSerErr
}

func (m *mockInventoryRepoCtrl) DeleteInventorySerial(id string) *responses.InternalResponse {
	return m.deleteSerErr
}

func (m *mockInventoryRepoCtrl) GetPickSuggestionsBySKU(sku string) ([]dto.PickSuggestion, *responses.InternalResponse) {
	return m.suggestions, m.suggestErr
}

func (m *mockInventoryRepoCtrl) GenerateImportTemplate(language string) ([]byte, error) {
	return []byte("tpl"), nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

const inventoryTestJWTSecret = "test-secret"

func generateInventoryTestToken(t *testing.T) string {
	t.Helper()
	token, err := tools.GenerateToken(inventoryTestJWTSecret, "user-1", "testuser", "test@example.com", "admin")
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return token
}

func newInventoryController(repo *mockInventoryRepoCtrl) *InventoryController {
	svc := services.NewInventoryService(repo, nil)
	return NewInventoryController(*svc, inventoryTestJWTSecret)
}

func performInventoryRequestWithToken(handler gin.HandlerFunc, method, path string, body interface{}, params gin.Params, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	c.Request = req
	if params != nil {
		c.Params = params
	}
	handler(c)
	return w
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestInventoryController_GetAllInventory_Empty(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{inventory: []*dto.EnhancedInventory{}})
	w := performRequest(ctrl.GetAllInventory, "GET", "/inventory", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_GetAllInventory_WithData(t *testing.T) {
	repo := &mockInventoryRepoCtrl{
		inventory: []*dto.EnhancedInventory{{ID: "inv-1", SKU: "SKU1", Location: "A-01"}},
	}
	ctrl := newInventoryController(repo)
	w := performRequest(ctrl.GetAllInventory, "GET", "/inventory", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_GetInventoryBySkuAndLocation_Found(t *testing.T) {
	repo := &mockInventoryRepoCtrl{
		bySkuLoc: map[string]*dto.EnhancedInventory{
			"SKU1:A-01": {ID: "inv-1", SKU: "SKU1", Location: "A-01"},
		},
	}
	ctrl := newInventoryController(repo)
	w := performRequest(ctrl.GetInventoryBySkuAndLocation, "GET", "/inventory/SKU1/A-01", nil,
		gin.Params{{Key: "sku", Value: "SKU1"}, {Key: "location", Value: "A-01"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_GetInventoryBySkuAndLocation_NotFound(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{bySkuLoc: map[string]*dto.EnhancedInventory{}})
	w := performRequest(ctrl.GetInventoryBySkuAndLocation, "GET", "/inventory/NOSKU/NOLOC", nil,
		gin.Params{{Key: "sku", Value: "NOSKU"}, {Key: "location", Value: "NOLOC"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInventoryController_GetInventoryBySkuAndLocation_MissingParam(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.GetInventoryBySkuAndLocation, "GET", "/inventory//", nil,
		gin.Params{{Key: "sku", Value: ""}, {Key: "location", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_CreateInventory_Success(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	token := generateInventoryTestToken(t)
	price := 10.0
	body := requests.CreateInventory{
		SKU:       "SKU-NEW",
		Name:      "New Item",
		Location:  "B-02",
		Quantity:  5,
		UnitPrice: &price,
	}
	w := performInventoryRequestWithToken(ctrl.CreateInventory, "POST", "/inventory", body, nil, token)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestInventoryController_CreateInventory_Unauthorized(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	price := 10.0
	body := requests.CreateInventory{
		SKU:       "SKU-NEW",
		Name:      "New Item",
		Location:  "B-02",
		Quantity:  5,
		UnitPrice: &price,
	}
	w := performInventoryRequestWithToken(ctrl.CreateInventory, "POST", "/inventory", body, nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInventoryController_CreateInventory_InvalidJSON(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	token := generateInventoryTestToken(t)
	w := performInventoryRequestWithToken(ctrl.CreateInventory, "POST", "/inventory", nil, nil, token)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_CreateInventory_Conflict(t *testing.T) {
	repo := &mockInventoryRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "inventory already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newInventoryController(repo)
	token := generateInventoryTestToken(t)
	price := 10.0
	body := requests.CreateInventory{
		SKU:       "DUP",
		Name:      "Dup Item",
		Location:  "A-01",
		Quantity:  1,
		UnitPrice: &price,
	}
	w := performInventoryRequestWithToken(ctrl.CreateInventory, "POST", "/inventory", body, nil, token)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestInventoryController_UpdateInventory_Success(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	price := 15.0
	body := requests.UpdateInventory{
		SKU:       "SKU1",
		Name:      "Updated Item",
		Location:  "A-01",
		Quantity:  10,
		UnitPrice: &price,
	}
	w := performRequest(ctrl.UpdateInventory, "PUT", "/inventory", body, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_UpdateInventory_NotFound(t *testing.T) {
	repo := &mockInventoryRepoCtrl{
		updateErr: &responses.InternalResponse{
			Message:    "inventory not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newInventoryController(repo)
	price := 15.0
	body := requests.UpdateInventory{
		SKU:       "NOSKU",
		Name:      "Ghost Item",
		Location:  "X-00",
		Quantity:  5.0,
		UnitPrice: &price,
	}
	w := performRequest(ctrl.UpdateInventory, "PUT", "/inventory", body, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInventoryController_DeleteInventory_Success(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.DeleteInventory, "DELETE", "/inventory/SKU1/A-01", nil,
		gin.Params{{Key: "id", Value: "SKU1"}, {Key: "location", Value: "A-01"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_DeleteInventory_NotFound(t *testing.T) {
	repo := &mockInventoryRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "inventory not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newInventoryController(repo)
	w := performRequest(ctrl.DeleteInventory, "DELETE", "/inventory/NOSKU/NOLOC", nil,
		gin.Params{{Key: "id", Value: "NOSKU"}, {Key: "location", Value: "NOLOC"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInventoryController_GetInventoryLots_Success(t *testing.T) {
	repo := &mockInventoryRepoCtrl{
		lots: []responses.InventoryLot{{ID: 1, InventoryID: 1, LotNumber: "L-001", Quantity: 5}},
	}
	ctrl := newInventoryController(repo)
	w := performRequest(ctrl.GetInventoryLots, "GET", "/inventory/inv-1/lots", nil,
		gin.Params{{Key: "id", Value: "inv-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_GetInventoryLots_MissingParam(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.GetInventoryLots, "GET", "/inventory//lots", nil,
		gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_GetInventorySerials_Success(t *testing.T) {
	repo := &mockInventoryRepoCtrl{
		serials: []responses.InventorySerialWithSerial{},
	}
	ctrl := newInventoryController(repo)
	w := performRequest(ctrl.GetInventorySerials, "GET", "/inventory/inv-1/serials", nil,
		gin.Params{{Key: "id", Value: "inv-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_GetInventorySerials_MissingParam(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.GetInventorySerials, "GET", "/inventory//serials", nil,
		gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_CreateInventoryLot_Success(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	body := requests.CreateInventoryLotRequest{
		LotID:    "lot-1",
		Quantity: 5,
		Location: "A-01",
	}
	w := performRequest(ctrl.CreateInventoryLot, "POST", "/inventory/inv-1/lots", body,
		gin.Params{{Key: "id", Value: "inv-1"}})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestInventoryController_CreateInventoryLot_MissingParam(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	body := requests.CreateInventoryLotRequest{LotID: "lot-1", Quantity: 5, Location: "A-01"}
	w := performRequest(ctrl.CreateInventoryLot, "POST", "/inventory//lots", body,
		gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_DeleteInventoryLot_Success(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.DeleteInventoryLot, "DELETE", "/inventory/lots/lot-1", nil,
		gin.Params{{Key: "id", Value: "lot-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_DeleteInventoryLot_MissingParam(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.DeleteInventoryLot, "DELETE", "/inventory/lots/", nil,
		gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_CreateInventorySerial_Success(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	body := requests.CreateInventorySerial{
		InventoryID: "inv-1",
		SerialID:    "ser-1",
		Location:    "A-01",
	}
	w := performRequest(ctrl.CreateInventorySerial, "POST", "/inventory/inv-1/serials", body,
		gin.Params{{Key: "id", Value: "inv-1"}})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestInventoryController_CreateInventorySerial_MissingParam(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	body := requests.CreateInventorySerial{InventoryID: "inv-1", SerialID: "ser-1", Location: "A-01"}
	w := performRequest(ctrl.CreateInventorySerial, "POST", "/inventory//serials", body,
		gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_DeleteInventorySerial_Success(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.DeleteInventorySerial, "DELETE", "/inventory/serials/ser-1", nil,
		gin.Params{{Key: "id", Value: "ser-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_DeleteInventorySerial_MissingParam(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.DeleteInventorySerial, "DELETE", "/inventory/serials/", nil,
		gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_GetPickSuggestions_Success(t *testing.T) {
	repo := &mockInventoryRepoCtrl{
		suggestions: []dto.PickSuggestion{{Location: "A-01", LotNumber: "L-001", Quantity: 3}},
	}
	ctrl := newInventoryController(repo)
	w := performRequest(ctrl.GetPickSuggestions, "GET", "/inventory/SKU1/pick", nil,
		gin.Params{{Key: "sku", Value: "SKU1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_GetPickSuggestions_MissingParam(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.GetPickSuggestions, "GET", "/inventory//pick", nil,
		gin.Params{{Key: "sku", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_ValidateImportRows_Success(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	rows := []requests.InventoryImportRow{
		{SKU: "SKU1", Name: "Item", Location: "A-01", Quantity: "5", UnitPrice: "10.0"},
	}
	w := performRequest(ctrl.ValidateImportRows, "POST", "/inventory/validate", rows, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryController_ValidateImportRows_EmptyBody(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.ValidateImportRows, "POST", "/inventory/validate", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInventoryController_ExportInventoryToExcel(t *testing.T) {
	ctrl := newInventoryController(&mockInventoryRepoCtrl{})
	w := performRequest(ctrl.ExportInventoryToExcel, "GET", "/inventory/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}
