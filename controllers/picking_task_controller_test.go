package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ───────────────────────────────────────────────────────────────

type mockPickingTaskRepoCtrl struct {
	tasks           []responses.PickingTaskView
	byID            map[string]*database.PickingTask
	createErr       *responses.InternalResponse
	startErr        *responses.InternalResponse
	updateErr       *responses.InternalResponse
	importErr       *responses.InternalResponse
	exportData      []byte
	exportErr       *responses.InternalResponse
	completeErr     *responses.InternalResponse
	completeLineErr *responses.InternalResponse
	templateData    []byte
	templateErr     error
}

func (m *mockPickingTaskRepoCtrl) GetAllPickingTasks() ([]responses.PickingTaskView, *responses.InternalResponse) {
	return m.tasks, nil
}

func (m *mockPickingTaskRepoCtrl) GetAllForTenant(tenantID string) ([]responses.PickingTaskView, *responses.InternalResponse) {
	return m.tasks, nil
}

func (m *mockPickingTaskRepoCtrl) GetPickingTaskByID(id string) (*database.PickingTask, *responses.InternalResponse) {
	if m.byID != nil {
		if t, ok := m.byID[id]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockPickingTaskRepoCtrl) CreatePickingTask(userId string, tenantID string, task *requests.CreatePickingTaskRequest) *responses.InternalResponse {
	return m.createErr
}

func (m *mockPickingTaskRepoCtrl) StartPickingTask(_ context.Context, id, userId string) *responses.InternalResponse {
	return m.startErr
}

func (m *mockPickingTaskRepoCtrl) UpdatePickingTask(_ context.Context, id string, data map[string]interface{}, userId string) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockPickingTaskRepoCtrl) ImportPickingTaskFromExcel(userID string, tenantID string, fileBytes []byte) *responses.InternalResponse {
	return m.importErr
}

func (m *mockPickingTaskRepoCtrl) ExportPickingTasksToExcel(_ string) ([]byte, *responses.InternalResponse) {
	return m.exportData, m.exportErr
}

func (m *mockPickingTaskRepoCtrl) CompletePickingTask(_ context.Context, id, userId string) *responses.InternalResponse {
	return m.completeErr
}

func (m *mockPickingTaskRepoCtrl) CompletePickingLine(_ context.Context, id, userId string, item requests.PickingTaskItemRequest) *responses.InternalResponse {
	return m.completeLineErr
}

func (m *mockPickingTaskRepoCtrl) GenerateImportTemplate(language string) ([]byte, error) {
	return m.templateData, m.templateErr
}

func (m *mockPickingTaskRepoCtrl) LinkCustomer(taskID string, customerID *string) *responses.InternalResponse {
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newPickingTasksController(repo *mockPickingTaskRepoCtrl) *PickingTasksController {
	svc := services.NewPickingTaskService(repo)
	return NewPickingTasksController(*svc, testJWTSecret)
}

func samplePickingTask() *database.PickingTask {
	items, _ := json.Marshal([]map[string]interface{}{})
	return &database.PickingTask{
		ID:          "pt-1",
		TaskID:      "TASK-001",
		OrderNumber: "ORD-001",
		CreatedBy:   "user-1",
		Status:      "open",
		Priority:    "normal",
		Items:       items,
	}
}

func samplePickingTaskRequest() requests.CreatePickingTaskRequest {
	items, _ := json.Marshal([]map[string]interface{}{{"sku": "SKU-001", "qty": 5}})
	return requests.CreatePickingTaskRequest{
		OutboundNumber: "OUT-001",
		Priority:       "normal",
		Items:          items,
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestPickingTasksController_GetAllPickingTasks_Empty(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{tasks: []responses.PickingTaskView{}})
	w := performRequest(ctrl.GetAllPickingTasks, "GET", "/picking-tasks", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_GetAllPickingTasks_WithData(t *testing.T) {
	repo := &mockPickingTaskRepoCtrl{
		tasks: []responses.PickingTaskView{{ID: "pt-1", TaskID: "TASK-001", Status: "open"}},
	}
	ctrl := newPickingTasksController(repo)
	w := performRequest(ctrl.GetAllPickingTasks, "GET", "/picking-tasks", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_GetPickingTaskByID_Found(t *testing.T) {
	task := samplePickingTask()
	repo := &mockPickingTaskRepoCtrl{byID: map[string]*database.PickingTask{"pt-1": task}}
	ctrl := newPickingTasksController(repo)
	w := performRequest(ctrl.GetPickingTaskByID, "GET", "/picking-tasks/pt-1", nil, gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_GetPickingTaskByID_NotFound(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{byID: map[string]*database.PickingTask{}})
	w := performRequest(ctrl.GetPickingTaskByID, "GET", "/picking-tasks/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPickingTasksController_GetPickingTaskByID_MissingParam(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequest(ctrl.GetPickingTaskByID, "GET", "/picking-tasks/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickingTasksController_CreatePickingTask_Success(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	body := samplePickingTaskRequest()
	w := performRequestWithHeader(ctrl.CreatePickingTask, "POST", "/picking-tasks", body, nil, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPickingTasksController_CreatePickingTask_InvalidJSON(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequest(ctrl.CreatePickingTask, "POST", "/picking-tasks", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickingTasksController_CreatePickingTask_Unauthorized(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	body := samplePickingTaskRequest()
	// No token
	w := performRequest(ctrl.CreatePickingTask, "POST", "/picking-tasks", body, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPickingTasksController_CreatePickingTask_ServiceError(t *testing.T) {
	repo := &mockPickingTaskRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newPickingTasksController(repo)
	body := samplePickingTaskRequest()
	w := performRequestWithHeader(ctrl.CreatePickingTask, "POST", "/picking-tasks", body, nil, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPickingTasksController_UpdatePickingTask_Success(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	body := map[string]interface{}{"priority": "high"}
	w := performRequestWithHeader(ctrl.UpdatePickingTask, "PUT", "/picking-tasks/pt-1", body,
		gin.Params{{Key: "id", Value: "pt-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_UpdatePickingTask_MissingParam(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	body := map[string]interface{}{"priority": "high"}
	w := performRequest(ctrl.UpdatePickingTask, "PUT", "/picking-tasks/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickingTasksController_UpdatePickingTask_NotFound(t *testing.T) {
	repo := &mockPickingTaskRepoCtrl{
		updateErr: &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound},
	}
	ctrl := newPickingTasksController(repo)
	body := map[string]interface{}{"priority": "high"}
	w := performRequestWithHeader(ctrl.UpdatePickingTask, "PUT", "/picking-tasks/99", body,
		gin.Params{{Key: "id", Value: "99"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPickingTasksController_StartPickingTask_Success(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequestWithHeader(ctrl.StartPickingTask, "PATCH", "/picking-tasks/pt-1/start", nil,
		gin.Params{{Key: "id", Value: "pt-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_StartPickingTask_Unauthorized(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequest(ctrl.StartPickingTask, "PATCH", "/picking-tasks/pt-1/start", nil,
		gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPickingTasksController_StartPickingTask_MissingParam(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequestWithHeader(ctrl.StartPickingTask, "PATCH", "/picking-tasks//start", nil,
		gin.Params{{Key: "id", Value: ""}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickingTasksController_CancelPickingTask_Success(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequestWithHeader(ctrl.CancelPickingTask, "PATCH", "/picking-tasks/pt-1/cancel", nil,
		gin.Params{{Key: "id", Value: "pt-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_CancelPickingTask_MissingParam(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequestWithHeader(ctrl.CancelPickingTask, "PATCH", "/picking-tasks//cancel", nil,
		gin.Params{{Key: "id", Value: ""}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickingTasksController_CompletePickingTask_Success(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequestWithHeader(ctrl.CompletePickingTask, "PATCH", "/picking-tasks/pt-1/complete", nil,
		gin.Params{{Key: "id", Value: "pt-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_CompletePickingTask_Unauthorized(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequest(ctrl.CompletePickingTask, "PATCH", "/picking-tasks/pt-1/complete", nil,
		gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPickingTasksController_CompletePickingTask_MissingParam(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequestWithHeader(ctrl.CompletePickingTask, "PATCH", "/picking-tasks//complete", nil,
		gin.Params{{Key: "id", Value: ""}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickingTasksController_CompletePickingLine_Success(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	body := requests.PickingTaskItemRequest{
		SKU:              "SKU-001",
		ExpectedQuantity: 5,
		Allocations: []database.LocationAllocation{
			{Location: "LOC-A", Quantity: 5},
		},
	}
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/picking-tasks/pt-1/complete-line", body,
		gin.Params{{Key: "id", Value: "pt-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_CompletePickingLine_Unauthorized(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	// Body must pass struct validation so the controller reaches the auth check.
	body := requests.PickingTaskItemRequest{
		SKU:              "SKU-001",
		ExpectedQuantity: 5,
		Allocations: []database.LocationAllocation{
			{Location: "LOC-A", Quantity: 5},
		},
	}
	w := performRequest(ctrl.CompletePickingLine, "PATCH", "/picking-tasks/pt-1/complete-line", body,
		gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPickingTasksController_CompletePickingLine_InvalidJSON(t *testing.T) {
	ctrl := newPickingTasksController(&mockPickingTaskRepoCtrl{})
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/picking-tasks/pt-1/complete-line", nil,
		gin.Params{{Key: "id", Value: "pt-1"}},
		map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPickingTasksController_ExportPickingTasksToExcel_Success(t *testing.T) {
	repo := &mockPickingTaskRepoCtrl{exportData: []byte("xlsx-data")}
	ctrl := newPickingTasksController(repo)
	w := performRequest(ctrl.ExportPickingTasksToExcel, "GET", "/picking-tasks/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPickingTasksController_ExportPickingTasksToExcel_Error(t *testing.T) {
	repo := &mockPickingTaskRepoCtrl{
		exportErr: &responses.InternalResponse{
			Message:    "export failed",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newPickingTasksController(repo)
	w := performRequest(ctrl.ExportPickingTasksToExcel, "GET", "/picking-tasks/export", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPickingTasksController_DownloadImportTemplate_Success(t *testing.T) {
	repo := &mockPickingTaskRepoCtrl{templateData: []byte("template-xlsx")}
	ctrl := newPickingTasksController(repo)
	w := performRequest(ctrl.DownloadImportTemplate, "GET", "/picking-tasks/template", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}
