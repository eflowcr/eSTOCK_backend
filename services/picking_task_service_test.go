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

// mockPickingTaskRepo is an in-memory fake for unit testing PickingTaskService.
type mockPickingTaskRepo struct {
	allTasks          []responses.PickingTaskView
	allTasksErr       *responses.InternalResponse
	byID              map[string]*database.PickingTask
	byIDErr           *responses.InternalResponse
	createErr         *responses.InternalResponse
	updateErr         *responses.InternalResponse
	importErr         *responses.InternalResponse
	exportBytes       []byte
	exportErr         *responses.InternalResponse
	completeTaskErr   *responses.InternalResponse
	completeLineErr   *responses.InternalResponse
	templateBytes     []byte
	templateErr       error
}

func (m *mockPickingTaskRepo) GetAllPickingTasks() ([]responses.PickingTaskView, *responses.InternalResponse) {
	return m.allTasks, m.allTasksErr
}

func (m *mockPickingTaskRepo) GetPickingTaskByID(id string) (*database.PickingTask, *responses.InternalResponse) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	if m.byID != nil {
		if t, ok := m.byID[id]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Picking task not found",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockPickingTaskRepo) CreatePickingTask(userId string, task *requests.CreatePickingTaskRequest) *responses.InternalResponse {
	return m.createErr
}

func (m *mockPickingTaskRepo) UpdatePickingTask(id string, data map[string]interface{}) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockPickingTaskRepo) ImportPickingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	return m.importErr
}

func (m *mockPickingTaskRepo) ExportPickingTasksToExcel() ([]byte, *responses.InternalResponse) {
	return m.exportBytes, m.exportErr
}

func (m *mockPickingTaskRepo) CompletePickingTask(id string, location, userId string) *responses.InternalResponse {
	return m.completeTaskErr
}

func (m *mockPickingTaskRepo) CompletePickingLine(id string, location, userId string, item requests.PickingTaskItemRequest) *responses.InternalResponse {
	return m.completeLineErr
}

func (m *mockPickingTaskRepo) GenerateImportTemplate(language string) ([]byte, error) {
	return m.templateBytes, m.templateErr
}

// --- Tests ---

func TestPickingTaskService_GetAllPickingTasks_Success(t *testing.T) {
	repo := &mockPickingTaskRepo{
		allTasks: []responses.PickingTaskView{
			{ID: "1", TaskID: "TASK-001", OrderNumber: "ORD-001", Status: "pending"},
			{ID: "2", TaskID: "TASK-002", OrderNumber: "ORD-002", Status: "completed"},
		},
	}
	svc := NewPickingTaskService(repo)
	list, errResp := svc.GetAllPickingTasks()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "TASK-001", list[0].TaskID)
	assert.Equal(t, "TASK-002", list[1].TaskID)
}

func TestPickingTaskService_GetAllPickingTasks_Error(t *testing.T) {
	repo := &mockPickingTaskRepo{
		allTasksErr: &responses.InternalResponse{
			Error:      errors.New("db error"),
			Message:    "Error fetching tasks",
			Handled:    false,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	svc := NewPickingTaskService(repo)
	list, errResp := svc.GetAllPickingTasks()
	require.NotNil(t, errResp)
	assert.Nil(t, list)
	assert.Equal(t, responses.StatusInternalServerError, errResp.StatusCode)
}

func TestPickingTaskService_GetPickingTaskByID_Found(t *testing.T) {
	repo := &mockPickingTaskRepo{
		byID: map[string]*database.PickingTask{
			"1": {ID: "1", TaskID: "TASK-001", OrderNumber: "ORD-001", Status: "pending"},
		},
	}
	svc := NewPickingTaskService(repo)
	task, errResp := svc.GetPickingTaskByID("1")
	require.Nil(t, errResp)
	require.NotNil(t, task)
	assert.Equal(t, "TASK-001", task.TaskID)
}

func TestPickingTaskService_GetPickingTaskByID_NotFound(t *testing.T) {
	repo := &mockPickingTaskRepo{byID: map[string]*database.PickingTask{}}
	svc := NewPickingTaskService(repo)
	task, errResp := svc.GetPickingTaskByID("99")
	require.NotNil(t, errResp)
	assert.Nil(t, task)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPickingTaskService_CreatePickingTask_Success(t *testing.T) {
	repo := &mockPickingTaskRepo{}
	svc := NewPickingTaskService(repo)
	req := &requests.CreatePickingTaskRequest{
		OutboundNumber: "ORD-001",
		Priority:       "normal",
	}
	errResp := svc.CreatePickingTask("user-1", req)
	require.Nil(t, errResp)
}

func TestPickingTaskService_CreatePickingTask_Error(t *testing.T) {
	repo := &mockPickingTaskRepo{
		createErr: &responses.InternalResponse{
			Message:    "Failed to create picking task",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewPickingTaskService(repo)
	req := &requests.CreatePickingTaskRequest{OutboundNumber: "ORD-DUP"}
	errResp := svc.CreatePickingTask("user-1", req)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestPickingTaskService_UpdatePickingTask_Success(t *testing.T) {
	repo := &mockPickingTaskRepo{}
	svc := NewPickingTaskService(repo)
	errResp := svc.UpdatePickingTask("1", map[string]interface{}{"status": "in_progress"})
	require.Nil(t, errResp)
}

func TestPickingTaskService_UpdatePickingTask_Error(t *testing.T) {
	repo := &mockPickingTaskRepo{
		updateErr: &responses.InternalResponse{
			Message:    "Task not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewPickingTaskService(repo)
	errResp := svc.UpdatePickingTask("99", map[string]interface{}{"status": "in_progress"})
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPickingTaskService_ImportPickingTaskFromExcel_Success(t *testing.T) {
	repo := &mockPickingTaskRepo{}
	svc := NewPickingTaskService(repo)
	errResp := svc.ImportPickingTaskFromExcel("user-1", []byte("data"))
	require.Nil(t, errResp)
}

func TestPickingTaskService_ImportPickingTaskFromExcel_Error(t *testing.T) {
	repo := &mockPickingTaskRepo{
		importErr: &responses.InternalResponse{
			Message:    "Invalid file format",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	svc := NewPickingTaskService(repo)
	errResp := svc.ImportPickingTaskFromExcel("user-1", []byte("bad"))
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
}

func TestPickingTaskService_ExportPickingTasksToExcel_Success(t *testing.T) {
	repo := &mockPickingTaskRepo{
		exportBytes: []byte("excel-data"),
	}
	svc := NewPickingTaskService(repo)
	data, errResp := svc.ExportPickingTasksToExcel()
	require.Nil(t, errResp)
	assert.Equal(t, []byte("excel-data"), data)
}

func TestPickingTaskService_ExportPickingTasksToExcel_Error(t *testing.T) {
	repo := &mockPickingTaskRepo{
		exportErr: &responses.InternalResponse{
			Error:   errors.New("export failed"),
			Message: "Export failed",
			Handled: false,
		},
	}
	svc := NewPickingTaskService(repo)
	data, errResp := svc.ExportPickingTasksToExcel()
	require.NotNil(t, errResp)
	assert.Nil(t, data)
}

func TestPickingTaskService_CompletePickingTask_Success(t *testing.T) {
	repo := &mockPickingTaskRepo{}
	svc := NewPickingTaskService(repo)
	errResp := svc.CompletePickingTask("1", "LOC-A", "user-1")
	require.Nil(t, errResp)
}

func TestPickingTaskService_CompletePickingTask_Error(t *testing.T) {
	repo := &mockPickingTaskRepo{
		completeTaskErr: &responses.InternalResponse{
			Message:    "Task already completed",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	svc := NewPickingTaskService(repo)
	errResp := svc.CompletePickingTask("1", "LOC-A", "user-1")
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
}

func TestPickingTaskService_CompletePickingLine_Success(t *testing.T) {
	repo := &mockPickingTaskRepo{}
	svc := NewPickingTaskService(repo)
	item := requests.PickingTaskItemRequest{
		SKU:              "SKU-001",
		ExpectedQuantity: 10,
		Location:         "LOC-A",
	}
	errResp := svc.CompletePickingLine("1", "LOC-A", "user-1", item)
	require.Nil(t, errResp)
}

func TestPickingTaskService_CompletePickingLine_Error(t *testing.T) {
	repo := &mockPickingTaskRepo{
		completeLineErr: &responses.InternalResponse{
			Message:    "Line not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewPickingTaskService(repo)
	item := requests.PickingTaskItemRequest{SKU: "SKU-001", Location: "LOC-A"}
	errResp := svc.CompletePickingLine("99", "LOC-A", "user-1", item)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestPickingTaskService_GenerateImportTemplate_Success(t *testing.T) {
	repo := &mockPickingTaskRepo{
		templateBytes: []byte("template-data"),
	}
	svc := NewPickingTaskService(repo)
	data, err := svc.GenerateImportTemplate("es")
	require.NoError(t, err)
	assert.Equal(t, []byte("template-data"), data)
}

func TestPickingTaskService_GenerateImportTemplate_Error(t *testing.T) {
	repo := &mockPickingTaskRepo{
		templateErr: errors.New("unsupported language"),
	}
	svc := NewPickingTaskService(repo)
	data, err := svc.GenerateImportTemplate("xx")
	require.Error(t, err)
	assert.Nil(t, data)
}
