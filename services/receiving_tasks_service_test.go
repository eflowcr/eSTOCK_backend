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

// mockReceivingTasksRepo is an in-memory fake for unit testing ReceivingTasksService.
type mockReceivingTasksRepo struct {
	allTasks        []responses.ReceivingTasksView
	allTasksErr     *responses.InternalResponse
	byID            map[string]*database.ReceivingTask
	byIDErr         *responses.InternalResponse
	createErr       *responses.InternalResponse
	updateErr       *responses.InternalResponse
	importErr       *responses.InternalResponse
	exportBytes     []byte
	exportErr       *responses.InternalResponse
	completeTaskErr *responses.InternalResponse
	completeLineErr *responses.InternalResponse
	templateBytes   []byte
	templateErr     error
}

func (m *mockReceivingTasksRepo) GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse) {
	return m.allTasks, m.allTasksErr
}

func (m *mockReceivingTasksRepo) GetReceivingTaskByID(id string) (*database.ReceivingTask, *responses.InternalResponse) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	if m.byID != nil {
		if t, ok := m.byID[id]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Receiving task not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockReceivingTasksRepo) CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse {
	return m.createErr
}

func (m *mockReceivingTasksRepo) UpdateReceivingTask(id string, data map[string]interface{}) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockReceivingTasksRepo) ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	return m.importErr
}

func (m *mockReceivingTasksRepo) ExportReceivingTaskToExcel() ([]byte, *responses.InternalResponse) {
	return m.exportBytes, m.exportErr
}

func (m *mockReceivingTasksRepo) CompleteFullTask(id string, location, userId string) *responses.InternalResponse {
	return m.completeTaskErr
}

func (m *mockReceivingTasksRepo) CompleteReceivingLine(id string, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse {
	return m.completeLineErr
}

func (m *mockReceivingTasksRepo) GenerateImportTemplate(language string) ([]byte, error) {
	return m.templateBytes, m.templateErr
}

func (m *mockReceivingTasksRepo) LinkSupplier(taskID string, supplierID *string) *responses.InternalResponse {
	return nil
}

// --- Tests ---

func TestReceivingTasksService_GetAllReceivingTasks_Success(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		allTasks: []responses.ReceivingTasksView{
			{ID: "1", TaskID: "RCV-001", InboundNumber: "INB-001", Status: "pending"},
			{ID: "2", TaskID: "RCV-002", InboundNumber: "INB-002", Status: "completed"},
		},
	}
	svc := NewReceivingTasksService(repo)
	list, errResp := svc.GetAllReceivingTasks()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "RCV-001", list[0].TaskID)
	assert.Equal(t, "RCV-002", list[1].TaskID)
}

func TestReceivingTasksService_GetAllReceivingTasks_Error(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		allTasksErr: &responses.InternalResponse{
			Error:      errors.New("db error"),
			Message:    "Error fetching tasks",
			Handled:    false,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	svc := NewReceivingTasksService(repo)
	list, errResp := svc.GetAllReceivingTasks()
	require.NotNil(t, errResp)
	assert.Nil(t, list)
	assert.Equal(t, responses.StatusInternalServerError, errResp.StatusCode)
}

func TestReceivingTasksService_GetReceivingTaskByID_Found(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		byID: map[string]*database.ReceivingTask{
			"1": {ID: "1", TaskID: "RCV-001", InboundNumber: "INB-001", Status: "pending"},
		},
	}
	svc := NewReceivingTasksService(repo)
	task, errResp := svc.GetReceivingTaskByID("1")
	require.Nil(t, errResp)
	require.NotNil(t, task)
	assert.Equal(t, "RCV-001", task.TaskID)
}

func TestReceivingTasksService_GetReceivingTaskByID_NotFound(t *testing.T) {
	repo := &mockReceivingTasksRepo{byID: map[string]*database.ReceivingTask{}}
	svc := NewReceivingTasksService(repo)
	task, errResp := svc.GetReceivingTaskByID("99")
	require.NotNil(t, errResp)
	assert.Nil(t, task)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestReceivingTasksService_CreateReceivingTask_Success(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	svc := NewReceivingTasksService(repo)
	req := &requests.CreateReceivingTaskRequest{
		InboundNumber: "INB-001",
		Priority:      "normal",
	}
	errResp := svc.CreateReceivingTask("user-1", req)
	require.Nil(t, errResp)
}

func TestReceivingTasksService_CreateReceivingTask_Error(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		createErr: &responses.InternalResponse{
			Message:    "Failed to create receiving task",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewReceivingTasksService(repo)
	req := &requests.CreateReceivingTaskRequest{InboundNumber: "INB-DUP"}
	errResp := svc.CreateReceivingTask("user-1", req)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestReceivingTasksService_UpdateReceivingTask_Success(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	svc := NewReceivingTasksService(repo)
	errResp := svc.UpdateReceivingTask("1", map[string]interface{}{"status": "in_progress"})
	require.Nil(t, errResp)
}

func TestReceivingTasksService_UpdateReceivingTask_Error(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		updateErr: &responses.InternalResponse{
			Message:    "Task not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewReceivingTasksService(repo)
	errResp := svc.UpdateReceivingTask("99", map[string]interface{}{"status": "in_progress"})
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestReceivingTasksService_ImportReceivingTaskFromExcel_Success(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	svc := NewReceivingTasksService(repo)
	errResp := svc.ImportReceivingTaskFromExcel("user-1", []byte("data"))
	require.Nil(t, errResp)
}

func TestReceivingTasksService_ImportReceivingTaskFromExcel_Error(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		importErr: &responses.InternalResponse{
			Message:    "Invalid file format",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	svc := NewReceivingTasksService(repo)
	errResp := svc.ImportReceivingTaskFromExcel("user-1", []byte("bad"))
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
}

func TestReceivingTasksService_ExportReceivingTaskToExcel_Success(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		exportBytes: []byte("excel-data"),
	}
	svc := NewReceivingTasksService(repo)
	data, errResp := svc.ExportReceivingTaskToExcel()
	require.Nil(t, errResp)
	assert.Equal(t, []byte("excel-data"), data)
}

func TestReceivingTasksService_ExportReceivingTaskToExcel_Error(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		exportErr: &responses.InternalResponse{
			Error:   errors.New("export failed"),
			Message: "Export failed",
			Handled: false,
		},
	}
	svc := NewReceivingTasksService(repo)
	data, errResp := svc.ExportReceivingTaskToExcel()
	require.NotNil(t, errResp)
	assert.Nil(t, data)
}

func TestReceivingTasksService_CompleteFullTask_Success(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	svc := NewReceivingTasksService(repo)
	errResp := svc.CompleteFullTask("1", "LOC-A", "user-1")
	require.Nil(t, errResp)
}

func TestReceivingTasksService_CompleteFullTask_Error(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		completeTaskErr: &responses.InternalResponse{
			Message:    "Task already completed",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	svc := NewReceivingTasksService(repo)
	errResp := svc.CompleteFullTask("1", "LOC-A", "user-1")
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
}

func TestReceivingTasksService_CompleteReceivingLine_Success(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	svc := NewReceivingTasksService(repo)
	item := requests.ReceivingTaskItemRequest{
		SKU:              "SKU-001",
		ExpectedQuantity: 10,
		Location:         "LOC-A",
	}
	errResp := svc.CompleteReceivingLine("1", "LOC-A", "user-1", item)
	require.Nil(t, errResp)
}

func TestReceivingTasksService_CompleteReceivingLine_Error(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		completeLineErr: &responses.InternalResponse{
			Message:    "Line not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewReceivingTasksService(repo)
	item := requests.ReceivingTaskItemRequest{SKU: "SKU-001", Location: "LOC-A"}
	errResp := svc.CompleteReceivingLine("99", "LOC-A", "user-1", item)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestReceivingTasksService_GenerateImportTemplate_Success(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		templateBytes: []byte("template-data"),
	}
	svc := NewReceivingTasksService(repo)
	data, err := svc.GenerateImportTemplate("es")
	require.NoError(t, err)
	assert.Equal(t, []byte("template-data"), data)
}

func TestReceivingTasksService_GenerateImportTemplate_Error(t *testing.T) {
	repo := &mockReceivingTasksRepo{
		templateErr: errors.New("unsupported language"),
	}
	svc := NewReceivingTasksService(repo)
	data, err := svc.GenerateImportTemplate("xx")
	require.Error(t, err)
	assert.Nil(t, data)
}

// ─────────────────────────────────────────────────────────────────────────────
// R1 — accepted/rejected backfill
// ─────────────────────────────────────────────────────────────────────────────

func TestApplyAcceptedRejectedBackfill_BothNil_ReceivedSet(t *testing.T) {
	// Legacy call: only received_qty set — accepted should be backfilled.
	rv := 50
	item := requests.ReceivingTaskItemRequest{
		SKU:              "SKU-1",
		ReceivedQuantity: &rv,
	}
	result := applyAcceptedRejectedBackfill(item)
	require.NotNil(t, result.AcceptedQty)
	assert.Equal(t, float64(50), *result.AcceptedQty)
}

func TestApplyAcceptedRejectedBackfill_AcceptedAlreadySet_NoBackfill(t *testing.T) {
	rv := 50
	accepted := 30.0
	item := requests.ReceivingTaskItemRequest{
		SKU:              "SKU-1",
		ReceivedQuantity: &rv,
		AcceptedQty:      &accepted,
	}
	result := applyAcceptedRejectedBackfill(item)
	require.NotNil(t, result.AcceptedQty)
	assert.Equal(t, 30.0, *result.AcceptedQty)
}

func TestApplyAcceptedRejectedBackfill_ReceivedNil_NoBackfill(t *testing.T) {
	item := requests.ReceivingTaskItemRequest{SKU: "SKU-1"}
	result := applyAcceptedRejectedBackfill(item)
	assert.Nil(t, result.AcceptedQty)
}

func TestReceivingTasksService_CompleteReceivingLine_LegacyBackfill(t *testing.T) {
	// Legacy request with only received_qty — service should backfill and call repo.
	rv := 20
	calledWithItem := &requests.ReceivingTaskItemRequest{}
	repo := &mockReceivingTasksRepoCapture{captured: calledWithItem}
	svc := NewReceivingTasksService(repo)

	item := requests.ReceivingTaskItemRequest{
		SKU:              "SKU-1",
		Location:         "LOC-A",
		ReceivedQuantity: &rv,
	}
	errResp := svc.CompleteReceivingLine("task-1", "LOC-A", "user-1", item)
	require.Nil(t, errResp)
	require.NotNil(t, calledWithItem.AcceptedQty)
	assert.Equal(t, float64(20), *calledWithItem.AcceptedQty)
}

// ─────────────────────────────────────────────────────────────────────────────
// R2 — supplier/customer validation
// ─────────────────────────────────────────────────────────────────────────────

func TestReceivingTasksService_LinkSupplier_ValidSupplier(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	cs := &mockClientsServiceForReceiving{
		client: &database.Client{ID: "c1", Type: "supplier"},
	}
	svc := NewReceivingTasksService(repo).WithClientsService(cs)
	supplierID := "c1"
	resp := svc.LinkSupplier("task-1", &supplierID)
	require.Nil(t, resp)
}

func TestReceivingTasksService_LinkSupplier_WrongType_Returns400(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	cs := &mockClientsServiceForReceiving{
		client: &database.Client{ID: "c1", Type: "customer"},
	}
	svc := NewReceivingTasksService(repo).WithClientsService(cs)
	supplierID := "c1"
	resp := svc.LinkSupplier("task-1", &supplierID)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

func TestReceivingTasksService_LinkSupplier_Unlink(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	svc := NewReceivingTasksService(repo)
	resp := svc.LinkSupplier("task-1", nil)
	require.Nil(t, resp)
}

func TestReceivingTasksService_CreateReceivingTask_InvalidSupplier_Returns400(t *testing.T) {
	repo := &mockReceivingTasksRepo{}
	cs := &mockClientsServiceForReceiving{
		client: &database.Client{ID: "c1", Type: "customer"},
	}
	svc := NewReceivingTasksService(repo).WithClientsService(cs)
	supplierID := "c1"
	req := &requests.CreateReceivingTaskRequest{
		InboundNumber: "INB-001",
		SupplierID:    &supplierID,
	}
	resp := svc.CreateReceivingTask("user-1", req)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test helpers
// ─────────────────────────────────────────────────────────────────────────────

// mockReceivingTasksRepoCapture captures the item passed to CompleteReceivingLine.
type mockReceivingTasksRepoCapture struct {
	captured *requests.ReceivingTaskItemRequest
}

func (m *mockReceivingTasksRepoCapture) GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockReceivingTasksRepoCapture) GetReceivingTaskByID(id string) (*database.ReceivingTask, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockReceivingTasksRepoCapture) CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse {
	return nil
}
func (m *mockReceivingTasksRepoCapture) UpdateReceivingTask(id string, data map[string]interface{}) *responses.InternalResponse {
	return nil
}
func (m *mockReceivingTasksRepoCapture) ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	return nil
}
func (m *mockReceivingTasksRepoCapture) ExportReceivingTaskToExcel() ([]byte, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockReceivingTasksRepoCapture) CompleteFullTask(id string, location, userId string) *responses.InternalResponse {
	return nil
}
func (m *mockReceivingTasksRepoCapture) CompleteReceivingLine(id string, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse {
	*m.captured = item
	return nil
}
func (m *mockReceivingTasksRepoCapture) GenerateImportTemplate(language string) ([]byte, error) {
	return nil, nil
}
func (m *mockReceivingTasksRepoCapture) LinkSupplier(taskID string, supplierID *string) *responses.InternalResponse {
	return nil
}

// mockClientsServiceForReceiving is a minimal fake ClientsService for tests.
type mockClientsServiceForReceiving struct {
	client  *database.Client
	getErr  *responses.InternalResponse
}

func (m *mockClientsServiceForReceiving) GetByID(id string) (*database.Client, *responses.InternalResponse) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.client, nil
}
