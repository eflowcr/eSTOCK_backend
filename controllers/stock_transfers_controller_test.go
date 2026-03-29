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
	"github.com/stretchr/testify/require"
)

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockStockTransfersRepoCtrl struct {
	transfers    []database.StockTransfer
	byID         map[string]*database.StockTransfer
	lines        []database.StockTransferLine
	byLineID     map[string]*database.StockTransferLine
	createErr    *responses.InternalResponse
	updateErr    *responses.InternalResponse
	deleteErr    *responses.InternalResponse
	createLineErr *responses.InternalResponse
	updateLineErr *responses.InternalResponse
	deleteLineErr *responses.InternalResponse
}

func (m *mockStockTransfersRepoCtrl) ListStockTransfers(status string) ([]database.StockTransfer, *responses.InternalResponse) {
	return m.transfers, nil
}

func (m *mockStockTransfersRepoCtrl) GetStockTransferByID(id string) (*database.StockTransfer, *responses.InternalResponse) {
	if m.byID != nil {
		if t, ok := m.byID[id]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockStockTransfersRepoCtrl) GetStockTransferByTransferNumber(transferNumber string) (*database.StockTransfer, *responses.InternalResponse) {
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockStockTransfersRepoCtrl) CreateStockTransfer(req *requests.StockTransferCreate, createdBy string) (*database.StockTransfer, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &database.StockTransfer{ID: "st-new", Status: "draft"}, nil
}

func (m *mockStockTransfersRepoCtrl) UpdateStockTransfer(id string, req *requests.StockTransferUpdate) (*database.StockTransfer, *responses.InternalResponse) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.byID != nil {
		if t, ok := m.byID[id]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockStockTransfersRepoCtrl) UpdateStockTransferStatus(id string, status string) (*database.StockTransfer, *responses.InternalResponse) {
	if m.byID != nil {
		if t, ok := m.byID[id]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockStockTransfersRepoCtrl) DeleteStockTransfer(id string) *responses.InternalResponse {
	return m.deleteErr
}

func (m *mockStockTransfersRepoCtrl) ListStockTransferLines(transferID string) ([]database.StockTransferLine, *responses.InternalResponse) {
	return m.lines, nil
}

func (m *mockStockTransfersRepoCtrl) CreateStockTransferLine(transferID string, req *requests.StockTransferLineInput) (*database.StockTransferLine, *responses.InternalResponse) {
	if m.createLineErr != nil {
		return nil, m.createLineErr
	}
	return &database.StockTransferLine{ID: "line-new", StockTransferID: transferID}, nil
}

func (m *mockStockTransfersRepoCtrl) UpdateStockTransferLine(lineID string, req *requests.StockTransferLineUpdate) (*database.StockTransferLine, *responses.InternalResponse) {
	if m.updateLineErr != nil {
		return nil, m.updateLineErr
	}
	if m.byLineID != nil {
		if l, ok := m.byLineID[lineID]; ok {
			return l, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockStockTransfersRepoCtrl) DeleteStockTransferLine(lineID string) *responses.InternalResponse {
	return m.deleteLineErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newStockTransfersController(repo *mockStockTransfersRepoCtrl) *StockTransfersController {
	svc := services.NewStockTransfersService(repo)
	return NewStockTransfersController(*svc, testJWTSecret, nil)
}

func marshalBody(body interface{}) *bytes.Buffer {
	if body == nil {
		return bytes.NewBuffer(nil)
	}
	b, _ := json.Marshal(body)
	return bytes.NewBuffer(b)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestStockTransfersController_ListStockTransfers_Empty(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{transfers: []database.StockTransfer{}})
	w := performRequest(ctrl.ListStockTransfers, "GET", "/stock-transfers", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockTransfersController_ListStockTransfers_WithData(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		transfers: []database.StockTransfer{{ID: "st-1", TransferNumber: "TF-001", Status: "draft"}},
	}
	ctrl := newStockTransfersController(repo)
	w := performRequest(ctrl.ListStockTransfers, "GET", "/stock-transfers", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Result.Success)
}

func TestStockTransfersController_GetStockTransferByID_Found(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		byID: map[string]*database.StockTransfer{
			"st-1": {ID: "st-1", TransferNumber: "TF-001", Status: "draft"},
		},
	}
	ctrl := newStockTransfersController(repo)
	w := performRequest(ctrl.GetStockTransferByID, "GET", "/stock-transfers/st-1", nil, gin.Params{{Key: "id", Value: "st-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockTransfersController_GetStockTransferByID_NotFound(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{byID: map[string]*database.StockTransfer{}})
	w := performRequest(ctrl.GetStockTransferByID, "GET", "/stock-transfers/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStockTransfersController_GetStockTransferByID_MissingParam(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequest(ctrl.GetStockTransferByID, "GET", "/stock-transfers/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_CreateStockTransfer_Success(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	body := requests.StockTransferCreate{
		FromLocationID: "loc-1",
		ToLocationID:   "loc-2",
		Lines: []requests.StockTransferLineInput{
			{Sku: "SKU-001", Quantity: 10},
		},
	}
	w := performRequestWithHeader(ctrl.CreateStockTransfer, "POST", "/stock-transfers", body, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestStockTransfersController_CreateStockTransfer_Unauthorized(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	body := requests.StockTransferCreate{
		FromLocationID: "loc-1",
		ToLocationID:   "loc-2",
		Lines:          []requests.StockTransferLineInput{{Sku: "SKU-001", Quantity: 10}},
	}
	w := performRequest(ctrl.CreateStockTransfer, "POST", "/stock-transfers", body, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestStockTransfersController_CreateStockTransfer_InvalidJSON(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequestWithHeader(ctrl.CreateStockTransfer, "POST", "/stock-transfers", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_CreateStockTransfer_ServiceError(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "conflict",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newStockTransfersController(repo)
	body := requests.StockTransferCreate{
		FromLocationID: "loc-1",
		ToLocationID:   "loc-2",
		Lines:          []requests.StockTransferLineInput{{Sku: "SKU-001", Quantity: 10}},
	}
	w := performRequestWithHeader(ctrl.CreateStockTransfer, "POST", "/stock-transfers", body, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestStockTransfersController_UpdateStockTransfer_Success(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		byID: map[string]*database.StockTransfer{
			"st-1": {ID: "st-1", TransferNumber: "TF-001", Status: "draft"},
		},
	}
	ctrl := newStockTransfersController(repo)
	body := requests.StockTransferUpdate{
		FromLocationID: "loc-1",
		ToLocationID:   "loc-2",
		Status:         "in_progress",
	}
	w := performRequest(ctrl.UpdateStockTransfer, "PUT", "/stock-transfers/st-1", body, gin.Params{{Key: "id", Value: "st-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockTransfersController_UpdateStockTransfer_MissingParam(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	body := requests.StockTransferUpdate{FromLocationID: "loc-1", ToLocationID: "loc-2", Status: "in_progress"}
	w := performRequest(ctrl.UpdateStockTransfer, "PUT", "/stock-transfers/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_UpdateStockTransfer_NotFound(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		updateErr: &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound},
	}
	ctrl := newStockTransfersController(repo)
	body := requests.StockTransferUpdate{FromLocationID: "loc-1", ToLocationID: "loc-2", Status: "in_progress"}
	w := performRequest(ctrl.UpdateStockTransfer, "PUT", "/stock-transfers/st-99", body, gin.Params{{Key: "id", Value: "st-99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStockTransfersController_DeleteStockTransfer_Success(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		byID: map[string]*database.StockTransfer{"st-1": {ID: "st-1"}},
	}
	ctrl := newStockTransfersController(repo)
	w := performRequest(ctrl.DeleteStockTransfer, "DELETE", "/stock-transfers/st-1", nil, gin.Params{{Key: "id", Value: "st-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockTransfersController_DeleteStockTransfer_MissingParam(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequest(ctrl.DeleteStockTransfer, "DELETE", "/stock-transfers/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_DeleteStockTransfer_Error(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newStockTransfersController(repo)
	w := performRequest(ctrl.DeleteStockTransfer, "DELETE", "/stock-transfers/st-1", nil, gin.Params{{Key: "id", Value: "st-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStockTransfersController_ListStockTransferLines_Success(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		lines: []database.StockTransferLine{{ID: "line-1", StockTransferID: "st-1", Sku: "SKU-001", Quantity: 5}},
	}
	ctrl := newStockTransfersController(repo)
	w := performRequest(ctrl.ListStockTransferLines, "GET", "/stock-transfers/st-1/lines", nil, gin.Params{{Key: "id", Value: "st-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockTransfersController_ListStockTransferLines_MissingParam(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequest(ctrl.ListStockTransferLines, "GET", "/stock-transfers//lines", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_CreateStockTransferLine_Success(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	body := requests.StockTransferLineInput{Sku: "SKU-001", Quantity: 10}
	w := performRequest(ctrl.CreateStockTransferLine, "POST", "/stock-transfers/st-1/lines", body, gin.Params{{Key: "id", Value: "st-1"}})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestStockTransfersController_CreateStockTransferLine_MissingParam(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	body := requests.StockTransferLineInput{Sku: "SKU-001", Quantity: 10}
	w := performRequest(ctrl.CreateStockTransferLine, "POST", "/stock-transfers//lines", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_CreateStockTransferLine_InvalidJSON(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequest(ctrl.CreateStockTransferLine, "POST", "/stock-transfers/st-1/lines", nil, gin.Params{{Key: "id", Value: "st-1"}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_DeleteStockTransferLine_Success(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequest(ctrl.DeleteStockTransferLine, "DELETE", "/stock-transfers/st-1/lines/line-1", nil, gin.Params{
		{Key: "id", Value: "st-1"},
		{Key: "lineId", Value: "line-1"},
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockTransfersController_DeleteStockTransferLine_MissingParam(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequest(ctrl.DeleteStockTransferLine, "DELETE", "/stock-transfers/st-1/lines/", nil, gin.Params{
		{Key: "id", Value: "st-1"},
		{Key: "lineId", Value: ""},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_DeleteStockTransferLine_Error(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		deleteLineErr: &responses.InternalResponse{
			Message:    "not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newStockTransfersController(repo)
	w := performRequest(ctrl.DeleteStockTransferLine, "DELETE", "/stock-transfers/st-1/lines/line-99", nil, gin.Params{
		{Key: "id", Value: "st-1"},
		{Key: "lineId", Value: "line-99"},
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStockTransfersController_ExecuteStockTransfer_Unauthorized(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequest(ctrl.ExecuteStockTransfer, "POST", "/stock-transfers/st-1/execute", nil, gin.Params{{Key: "id", Value: "st-1"}})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestStockTransfersController_ExecuteStockTransfer_MissingParam(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	w := performRequestWithHeader(ctrl.ExecuteStockTransfer, "POST", "/stock-transfers//execute", nil, gin.Params{{Key: "id", Value: ""}}, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_ExecuteStockTransfer_ServiceError(t *testing.T) {
	// ExecuteTransfer without LocationsRepository/DB configured returns 500
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{
		byID: map[string]*database.StockTransfer{
			"st-1": {ID: "st-1", Status: "draft"},
		},
	})
	w := performRequestWithHeader(ctrl.ExecuteStockTransfer, "POST", "/stock-transfers/st-1/execute", nil, gin.Params{{Key: "id", Value: "st-1"}}, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStockTransfersController_UpdateStockTransferLine_Success(t *testing.T) {
	repo := &mockStockTransfersRepoCtrl{
		byLineID: map[string]*database.StockTransferLine{
			"line-1": {ID: "line-1", StockTransferID: "st-1"},
		},
	}
	ctrl := newStockTransfersController(repo)
	body := requests.StockTransferLineUpdate{Quantity: 5}
	w := performRequest(ctrl.UpdateStockTransferLine, "PUT", "/stock-transfers/st-1/lines/line-1", body, gin.Params{
		{Key: "id", Value: "st-1"},
		{Key: "lineId", Value: "line-1"},
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStockTransfersController_UpdateStockTransferLine_MissingParam(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})
	body := requests.StockTransferLineUpdate{Quantity: 5}
	w := performRequest(ctrl.UpdateStockTransferLine, "PUT", "/stock-transfers/st-1/lines/", body, gin.Params{
		{Key: "id", Value: "st-1"},
		{Key: "lineId", Value: ""},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStockTransfersController_UpdateStockTransferLine_InvalidJSON(t *testing.T) {
	ctrl := newStockTransfersController(&mockStockTransfersRepoCtrl{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("PUT", "/stock-transfers/st-1/lines/line-1", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "st-1"}, {Key: "lineId", Value: "line-1"}}
	ctrl.UpdateStockTransferLine(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
