package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockReceivingTasksRepoCtrl struct {
	tasks         []responses.ReceivingTasksView
	byID          map[string]*database.ReceivingTask
	createErr     *responses.InternalResponse
	updateErr     *responses.InternalResponse
	importErr     *responses.InternalResponse
	exportData    []byte
	exportErr     *responses.InternalResponse
	completeErr   *responses.InternalResponse
	completeLineErr *responses.InternalResponse
	templateErr   error
}

func (m *mockReceivingTasksRepoCtrl) GetAllReceivingTasks() ([]responses.ReceivingTasksView, *responses.InternalResponse) {
	return m.tasks, nil
}

func (m *mockReceivingTasksRepoCtrl) GetReceivingTaskByID(id string) (*database.ReceivingTask, *responses.InternalResponse) {
	if m.byID != nil {
		if t, ok := m.byID[id]; ok {
			return t, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockReceivingTasksRepoCtrl) CreateReceivingTask(userId string, task *requests.CreateReceivingTaskRequest) *responses.InternalResponse {
	return m.createErr
}

func (m *mockReceivingTasksRepoCtrl) UpdateReceivingTask(id string, data map[string]interface{}) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockReceivingTasksRepoCtrl) ImportReceivingTaskFromExcel(userID string, fileBytes []byte) *responses.InternalResponse {
	return m.importErr
}

func (m *mockReceivingTasksRepoCtrl) ExportReceivingTaskToExcel() ([]byte, *responses.InternalResponse) {
	return m.exportData, m.exportErr
}

func (m *mockReceivingTasksRepoCtrl) CompleteFullTask(id string, location, userId string) *responses.InternalResponse {
	return m.completeErr
}

func (m *mockReceivingTasksRepoCtrl) CompleteReceivingLine(id string, location, userId string, item requests.ReceivingTaskItemRequest) *responses.InternalResponse {
	return m.completeLineErr
}

func (m *mockReceivingTasksRepoCtrl) GenerateImportTemplate(language string) ([]byte, error) {
	if m.templateErr != nil {
		return nil, m.templateErr
	}
	return []byte("template"), nil
}

func (m *mockReceivingTasksRepoCtrl) LinkSupplier(taskID string, supplierID *string) *responses.InternalResponse {
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

const testJWTSecret = "test-secret"

func makeTestToken() string {
	token, _ := tools.GenerateToken(testJWTSecret, "user-1", "testuser", "test@test.com", "admin")
	return "Bearer " + token
}

func newReceivingTasksController(repo *mockReceivingTasksRepoCtrl) *ReceivingTasksController {
	svc := services.NewReceivingTasksService(repo)
	return NewReceivingTasksController(*svc, testJWTSecret)
}

func performRequestWithHeader(handler gin.HandlerFunc, method, path string, body interface{}, params gin.Params, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(method, path, marshalBody(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	if params != nil {
		c.Params = params
	}
	handler(c)
	return w
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestReceivingTasksController_GetAllReceivingTasks_Empty(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{tasks: []responses.ReceivingTasksView{}})
	w := performRequest(ctrl.GetAllReceivingTasks, "GET", "/receiving-tasks", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReceivingTasksController_GetAllReceivingTasks_WithData(t *testing.T) {
	repo := &mockReceivingTasksRepoCtrl{
		tasks: []responses.ReceivingTasksView{{ID: "t-1", TaskID: "TASK-001", InboundNumber: "IB-001"}},
	}
	ctrl := newReceivingTasksController(repo)
	w := performRequest(ctrl.GetAllReceivingTasks, "GET", "/receiving-tasks", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Result.Success)
}

func TestReceivingTasksController_GetReceivingTaskByID_Found(t *testing.T) {
	repo := &mockReceivingTasksRepoCtrl{
		byID: map[string]*database.ReceivingTask{
			"t-1": {ID: "t-1", TaskID: "TASK-001", InboundNumber: "IB-001", Status: "pending", Priority: "normal"},
		},
	}
	ctrl := newReceivingTasksController(repo)
	w := performRequest(ctrl.GetReceivingTaskByID, "GET", "/receiving-tasks/t-1", nil, gin.Params{{Key: "id", Value: "t-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReceivingTasksController_GetReceivingTaskByID_NotFound(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{byID: map[string]*database.ReceivingTask{}})
	w := performRequest(ctrl.GetReceivingTaskByID, "GET", "/receiving-tasks/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReceivingTasksController_GetReceivingTaskByID_MissingParam(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	w := performRequest(ctrl.GetReceivingTaskByID, "GET", "/receiving-tasks/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReceivingTasksController_CreateReceivingTask_Success(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	body := requests.CreateReceivingTaskRequest{
		InboundNumber: "IB-100",
		Priority:      "normal",
		Items:         []byte(`[]`),
	}
	w := performRequestWithHeader(ctrl.CreateReceivingTask, "POST", "/receiving-tasks", body, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestReceivingTasksController_CreateReceivingTask_Unauthorized(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	body := requests.CreateReceivingTaskRequest{
		InboundNumber: "IB-100",
		Priority:      "normal",
		Items:         []byte(`[]`),
	}
	w := performRequest(ctrl.CreateReceivingTask, "POST", "/receiving-tasks", body, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestReceivingTasksController_CreateReceivingTask_InvalidJSON(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	w := performRequestWithHeader(ctrl.CreateReceivingTask, "POST", "/receiving-tasks", nil, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReceivingTasksController_CreateReceivingTask_ServiceError(t *testing.T) {
	repo := &mockReceivingTasksRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "db error",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newReceivingTasksController(repo)
	body := requests.CreateReceivingTaskRequest{
		InboundNumber: "IB-200",
		Priority:      "normal",
		Items:         []byte(`[]`),
	}
	w := performRequestWithHeader(ctrl.CreateReceivingTask, "POST", "/receiving-tasks", body, nil, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReceivingTasksController_UpdateReceivingTask_Success(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	body := map[string]interface{}{"status": "in_progress"}
	w := performRequest(ctrl.UpdateReceivingTask, "PATCH", "/receiving-tasks/t-1", body, gin.Params{{Key: "id", Value: "t-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReceivingTasksController_UpdateReceivingTask_MissingParam(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	body := map[string]interface{}{"status": "in_progress"}
	w := performRequest(ctrl.UpdateReceivingTask, "PATCH", "/receiving-tasks/", body, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReceivingTasksController_UpdateReceivingTask_ServiceError(t *testing.T) {
	repo := &mockReceivingTasksRepoCtrl{
		updateErr: &responses.InternalResponse{
			Message:    "not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newReceivingTasksController(repo)
	body := map[string]interface{}{"status": "completed"}
	w := performRequest(ctrl.UpdateReceivingTask, "PATCH", "/receiving-tasks/t-99", body, gin.Params{{Key: "id", Value: "t-99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReceivingTasksController_ExportReceivingTaskToExcel_Success(t *testing.T) {
	repo := &mockReceivingTasksRepoCtrl{exportData: []byte("xlsx")}
	ctrl := newReceivingTasksController(repo)
	w := performRequest(ctrl.ExportReceivingTaskToExcel, "GET", "/receiving-tasks/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReceivingTasksController_ExportReceivingTaskToExcel_Error(t *testing.T) {
	repo := &mockReceivingTasksRepoCtrl{
		exportErr: &responses.InternalResponse{
			Message:    "export failed",
			Handled:    true,
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newReceivingTasksController(repo)
	w := performRequest(ctrl.ExportReceivingTaskToExcel, "GET", "/receiving-tasks/export", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReceivingTasksController_DownloadImportTemplate_Success(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	w := performRequest(ctrl.DownloadImportTemplate, "GET", "/receiving-tasks/template", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReceivingTasksController_CompleteFullTask_Success(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	w := performRequestWithHeader(ctrl.CompleteFullTask, "POST", "/receiving-tasks/t-1/complete", nil, gin.Params{{Key: "id", Value: "t-1"}}, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReceivingTasksController_CompleteFullTask_MissingParam(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	w := performRequestWithHeader(ctrl.CompleteFullTask, "POST", "/receiving-tasks//complete", nil, gin.Params{{Key: "id", Value: ""}}, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReceivingTasksController_CompleteFullTask_Unauthorized(t *testing.T) {
	ctrl := newReceivingTasksController(&mockReceivingTasksRepoCtrl{})
	w := performRequest(ctrl.CompleteFullTask, "POST", "/receiving-tasks/t-1/complete", nil, gin.Params{{Key: "id", Value: "t-1"}})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestReceivingTasksController_CompleteFullTask_ServiceError(t *testing.T) {
	repo := &mockReceivingTasksRepoCtrl{
		completeErr: &responses.InternalResponse{
			Message:    "not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newReceivingTasksController(repo)
	w := performRequestWithHeader(ctrl.CompleteFullTask, "POST", "/receiving-tasks/t-99/complete", nil, gin.Params{{Key: "id", Value: "t-99"}}, map[string]string{
		"Authorization": makeTestToken(),
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}
