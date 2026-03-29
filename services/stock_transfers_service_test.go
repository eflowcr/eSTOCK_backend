package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStockTransfersRepo is an in-memory fake for unit testing StockTransfersService.
type mockStockTransfersRepo struct {
	transfers           []database.StockTransfer
	listErr             *responses.InternalResponse
	byID                map[string]*database.StockTransfer
	byIDErr             *responses.InternalResponse
	byNumber            map[string]*database.StockTransfer
	byNumberErr         *responses.InternalResponse
	createResult        *database.StockTransfer
	createErr           *responses.InternalResponse
	updateResult        *database.StockTransfer
	updateErr           *responses.InternalResponse
	updateStatusResult  *database.StockTransfer
	updateStatusErr     *responses.InternalResponse
	deleteErr           *responses.InternalResponse
	lines               []database.StockTransferLine
	linesErr            *responses.InternalResponse
	createLineResult    *database.StockTransferLine
	createLineErr       *responses.InternalResponse
	updateLineResult    *database.StockTransferLine
	updateLineErr       *responses.InternalResponse
	deleteLineErr       *responses.InternalResponse
}

func (m *mockStockTransfersRepo) ListStockTransfers(status string) ([]database.StockTransfer, *responses.InternalResponse) {
	return m.transfers, m.listErr
}

func (m *mockStockTransfersRepo) GetStockTransferByID(id string) (*database.StockTransfer, *responses.InternalResponse) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	if m.byID != nil {
		if t, ok := m.byID[id]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Stock transfer not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockStockTransfersRepo) GetStockTransferByTransferNumber(transferNumber string) (*database.StockTransfer, *responses.InternalResponse) {
	if m.byNumberErr != nil {
		return nil, m.byNumberErr
	}
	if m.byNumber != nil {
		if t, ok := m.byNumber[transferNumber]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Stock transfer not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockStockTransfersRepo) CreateStockTransfer(req *requests.StockTransferCreate, createdBy string) (*database.StockTransfer, *responses.InternalResponse) {
	return m.createResult, m.createErr
}

func (m *mockStockTransfersRepo) UpdateStockTransfer(id string, req *requests.StockTransferUpdate) (*database.StockTransfer, *responses.InternalResponse) {
	return m.updateResult, m.updateErr
}

func (m *mockStockTransfersRepo) UpdateStockTransferStatus(id string, status string) (*database.StockTransfer, *responses.InternalResponse) {
	return m.updateStatusResult, m.updateStatusErr
}

func (m *mockStockTransfersRepo) DeleteStockTransfer(id string) *responses.InternalResponse {
	return m.deleteErr
}

func (m *mockStockTransfersRepo) ListStockTransferLines(transferID string) ([]database.StockTransferLine, *responses.InternalResponse) {
	return m.lines, m.linesErr
}

func (m *mockStockTransfersRepo) CreateStockTransferLine(transferID string, req *requests.StockTransferLineInput) (*database.StockTransferLine, *responses.InternalResponse) {
	return m.createLineResult, m.createLineErr
}

func (m *mockStockTransfersRepo) UpdateStockTransferLine(lineID string, req *requests.StockTransferLineUpdate) (*database.StockTransferLine, *responses.InternalResponse) {
	return m.updateLineResult, m.updateLineErr
}

func (m *mockStockTransfersRepo) DeleteStockTransferLine(lineID string) *responses.InternalResponse {
	return m.deleteLineErr
}

// --- Tests ---

func TestStockTransfersService_ListStockTransfers_Success(t *testing.T) {
	repo := &mockStockTransfersRepo{
		transfers: []database.StockTransfer{
			{ID: "1", TransferNumber: "TRF-001", Status: "draft"},
			{ID: "2", TransferNumber: "TRF-002", Status: "completed"},
		},
	}
	svc := NewStockTransfersService(repo)
	list, errResp := svc.ListStockTransfers("")
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "TRF-001", list[0].TransferNumber)
}

func TestStockTransfersService_ListStockTransfers_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		listErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error listing transfers",
			Handled: false,
		},
	}
	svc := NewStockTransfersService(repo)
	list, errResp := svc.ListStockTransfers("")
	require.NotNil(t, errResp)
	assert.Nil(t, list)
}

func TestStockTransfersService_GetStockTransferByID_Found(t *testing.T) {
	repo := &mockStockTransfersRepo{
		byID: map[string]*database.StockTransfer{
			"1": {ID: "1", TransferNumber: "TRF-001", Status: "draft"},
		},
	}
	svc := NewStockTransfersService(repo)
	transfer, errResp := svc.GetStockTransferByID("1")
	require.Nil(t, errResp)
	require.NotNil(t, transfer)
	assert.Equal(t, "TRF-001", transfer.TransferNumber)
}

func TestStockTransfersService_GetStockTransferByID_NotFound(t *testing.T) {
	repo := &mockStockTransfersRepo{byID: map[string]*database.StockTransfer{}}
	svc := NewStockTransfersService(repo)
	transfer, errResp := svc.GetStockTransferByID("99")
	require.NotNil(t, errResp)
	assert.Nil(t, transfer)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestStockTransfersService_GetStockTransferByTransferNumber_Found(t *testing.T) {
	repo := &mockStockTransfersRepo{
		byNumber: map[string]*database.StockTransfer{
			"TRF-001": {ID: "1", TransferNumber: "TRF-001", Status: "draft"},
		},
	}
	svc := NewStockTransfersService(repo)
	transfer, errResp := svc.GetStockTransferByTransferNumber("TRF-001")
	require.Nil(t, errResp)
	require.NotNil(t, transfer)
	assert.Equal(t, "TRF-001", transfer.TransferNumber)
}

func TestStockTransfersService_GetStockTransferByTransferNumber_NotFound(t *testing.T) {
	repo := &mockStockTransfersRepo{byNumber: map[string]*database.StockTransfer{}}
	svc := NewStockTransfersService(repo)
	transfer, errResp := svc.GetStockTransferByTransferNumber("UNKNOWN")
	require.NotNil(t, errResp)
	assert.Nil(t, transfer)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestStockTransfersService_CreateStockTransfer_Success(t *testing.T) {
	expected := &database.StockTransfer{
		ID:             "1",
		TransferNumber: "TRF-001",
		FromLocationID: "loc-a",
		ToLocationID:   "loc-b",
		Status:         "draft",
	}
	repo := &mockStockTransfersRepo{createResult: expected}
	svc := NewStockTransfersService(repo)
	req := &requests.StockTransferCreate{
		FromLocationID: "loc-a",
		ToLocationID:   "loc-b",
	}
	result, errResp := svc.CreateStockTransfer(req, "user-1")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "TRF-001", result.TransferNumber)
}

func TestStockTransfersService_CreateStockTransfer_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		createErr: &responses.InternalResponse{
			Message:    "Failed to create transfer",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	svc := NewStockTransfersService(repo)
	req := &requests.StockTransferCreate{
		FromLocationID: "loc-a",
		ToLocationID:   "loc-b",
	}
	result, errResp := svc.CreateStockTransfer(req, "user-1")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
}

func TestStockTransfersService_UpdateStockTransfer_Success(t *testing.T) {
	expected := &database.StockTransfer{ID: "1", Status: "in_progress"}
	repo := &mockStockTransfersRepo{updateResult: expected}
	svc := NewStockTransfersService(repo)
	req := &requests.StockTransferUpdate{
		FromLocationID: "loc-a",
		ToLocationID:   "loc-b",
		Status:         "in_progress",
	}
	result, errResp := svc.UpdateStockTransfer("1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "in_progress", result.Status)
}

func TestStockTransfersService_UpdateStockTransfer_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		updateErr: &responses.InternalResponse{
			Message:    "Transfer not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewStockTransfersService(repo)
	req := &requests.StockTransferUpdate{FromLocationID: "loc-a", ToLocationID: "loc-b", Status: "in_progress"}
	result, errResp := svc.UpdateStockTransfer("99", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestStockTransfersService_UpdateStockTransferStatus_Success(t *testing.T) {
	expected := &database.StockTransfer{ID: "1", Status: "completed"}
	repo := &mockStockTransfersRepo{updateStatusResult: expected}
	svc := NewStockTransfersService(repo)
	result, errResp := svc.UpdateStockTransferStatus("1", "completed")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "completed", result.Status)
}

func TestStockTransfersService_UpdateStockTransferStatus_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		updateStatusErr: &responses.InternalResponse{
			Message:    "Transfer not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewStockTransfersService(repo)
	result, errResp := svc.UpdateStockTransferStatus("99", "completed")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
}

func TestStockTransfersService_DeleteStockTransfer_Success(t *testing.T) {
	repo := &mockStockTransfersRepo{}
	svc := NewStockTransfersService(repo)
	errResp := svc.DeleteStockTransfer("1")
	require.Nil(t, errResp)
}

func TestStockTransfersService_DeleteStockTransfer_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		deleteErr: &responses.InternalResponse{
			Message:    "Transfer not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewStockTransfersService(repo)
	errResp := svc.DeleteStockTransfer("99")
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestStockTransfersService_ListStockTransferLines_Success(t *testing.T) {
	repo := &mockStockTransfersRepo{
		lines: []database.StockTransferLine{
			{ID: "l1", StockTransferID: "1", Sku: "SKU-001", Quantity: 5},
			{ID: "l2", StockTransferID: "1", Sku: "SKU-002", Quantity: 10},
		},
	}
	svc := NewStockTransfersService(repo)
	lines, errResp := svc.ListStockTransferLines("1")
	require.Nil(t, errResp)
	require.Len(t, lines, 2)
	assert.Equal(t, "SKU-001", lines[0].Sku)
}

func TestStockTransfersService_ListStockTransferLines_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		linesErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error fetching lines",
			Handled: false,
		},
	}
	svc := NewStockTransfersService(repo)
	lines, errResp := svc.ListStockTransferLines("1")
	require.NotNil(t, errResp)
	assert.Nil(t, lines)
}

func TestStockTransfersService_CreateStockTransferLine_Success(t *testing.T) {
	expected := &database.StockTransferLine{ID: "l1", StockTransferID: "1", Sku: "SKU-001", Quantity: 5}
	repo := &mockStockTransfersRepo{createLineResult: expected}
	svc := NewStockTransfersService(repo)
	req := &requests.StockTransferLineInput{Sku: "SKU-001", Quantity: 5}
	result, errResp := svc.CreateStockTransferLine("1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "SKU-001", result.Sku)
}

func TestStockTransfersService_CreateStockTransferLine_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		createLineErr: &responses.InternalResponse{
			Message:    "Invalid SKU",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	svc := NewStockTransfersService(repo)
	req := &requests.StockTransferLineInput{Sku: "BAD-SKU", Quantity: 5}
	result, errResp := svc.CreateStockTransferLine("1", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
}

func TestStockTransfersService_UpdateStockTransferLine_Success(t *testing.T) {
	expected := &database.StockTransferLine{ID: "l1", Quantity: 20}
	repo := &mockStockTransfersRepo{updateLineResult: expected}
	svc := NewStockTransfersService(repo)
	req := &requests.StockTransferLineUpdate{Quantity: 20}
	result, errResp := svc.UpdateStockTransferLine("l1", req)
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, float64(20), result.Quantity)
}

func TestStockTransfersService_UpdateStockTransferLine_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		updateLineErr: &responses.InternalResponse{
			Message:    "Line not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewStockTransfersService(repo)
	req := &requests.StockTransferLineUpdate{Quantity: 20}
	result, errResp := svc.UpdateStockTransferLine("99", req)
	require.NotNil(t, errResp)
	assert.Nil(t, result)
}

func TestStockTransfersService_DeleteStockTransferLine_Success(t *testing.T) {
	repo := &mockStockTransfersRepo{}
	svc := NewStockTransfersService(repo)
	errResp := svc.DeleteStockTransferLine("l1")
	require.Nil(t, errResp)
}

func TestStockTransfersService_DeleteStockTransferLine_Error(t *testing.T) {
	repo := &mockStockTransfersRepo{
		deleteLineErr: &responses.InternalResponse{
			Message:    "Line not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewStockTransfersService(repo)
	errResp := svc.DeleteStockTransferLine("99")
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestStockTransfersService_ExecuteTransfer_MissingDeps(t *testing.T) {
	repo := &mockStockTransfersRepo{}
	// Use basic constructor — LocationsRepository and DB are nil
	svc := NewStockTransfersService(repo)
	result, errResp := svc.ExecuteTransfer("1", "user-1")
	require.NotNil(t, errResp)
	assert.Nil(t, result)
	assert.Equal(t, responses.StatusInternalServerError, errResp.StatusCode)
	assert.True(t, errResp.Handled)
}
