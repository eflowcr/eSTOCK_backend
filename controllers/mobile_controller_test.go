package controllers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimal stub services backing MobileController for smoke-level tests.

// reuse mockPickingTaskRepoCtrl from picking_task_controller_test.go (same package).
// reuse mockReceivingTasksRepoCtrl from receiving_tasks_controller_test.go.

// buildTaskItemsJSON returns the jsonb encoding of one or more items, panicking
// on marshaling errors (test code only).
func buildTaskItemsJSON(items ...requests.PickingTaskItemRequest) []byte {
	b, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}
	return b
}

// sampleTaskWithItem returns a PickingTask with a single item (SKU-1, expected 10,
// at LOC-A) for use in the picking-line tests. Mirrors the W0.7 contract: items
// is a jsonb of []PickingTaskItemRequest with allocations[0].location set.
func sampleTaskWithItem() *database.PickingTask {
	items := buildTaskItemsJSON(requests.PickingTaskItemRequest{
		SKU:              "SKU-1",
		ExpectedQuantity: 10,
		Allocations:      []database.LocationAllocation{{Location: "LOC-A", Quantity: 10}},
	})
	return &database.PickingTask{
		ID:     "pt-1",
		TaskID: "T1",
		Status: "in_progress",
		Items:  items,
	}
}

func newTestMobileController() *MobileController {
	pickRepo := &mockPickingTaskRepoCtrl{
		tasks: []responses.PickingTaskView{{ID: "pt-1", TaskID: "T1", OrderNumber: "O1", Status: "in_progress"}},
		byID:  map[string]*database.PickingTask{"pt-1": sampleTaskWithItem()},
	}
	pickSvc := services.NewPickingTaskService(pickRepo)
	recvRepo := &mockReceivingTasksRepoCtrl{
		tasks: []responses.ReceivingTasksView{},
		byID:  map[string]*database.ReceivingTask{},
	}
	recvSvc := services.NewReceivingTasksService(recvRepo)
	cfg := configuration.Config{JWTSecret: testJWTSecret, Version: "test"}
	return NewMobileController(pickSvc, recvSvc, nil, nil, nil, nil, cfg)
}

// ─── /api/mobile/health ──────────────────────────────────────────────────────

func TestMobileController_Health_Success(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequestWithHeader(ctrl.Health, "GET", "/api/mobile/health", nil, nil, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)

	var env responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.True(t, env.Result.Success)

	// data is map[string]interface{} after Unmarshal — check it has expected keys.
	dataMap, ok := env.Data.(map[string]interface{})
	require.True(t, ok, "data should be an object")
	assert.Equal(t, "user-1", dataMap["user"])
	assert.Equal(t, "admin", dataMap["role"])
	assert.Equal(t, "test", dataMap["version"])
}

func TestMobileController_Health_Unauthorized(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequest(ctrl.Health, "GET", "/api/mobile/health", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── /api/mobile/picking-tasks ───────────────────────────────────────────────

func TestMobileController_ListPickingTasks_Success(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequestWithHeader(ctrl.ListPickingTasks, "GET", "/api/mobile/picking-tasks", nil, nil, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMobileController_ListPickingTasks_FilterAssignedToMe(t *testing.T) {
	uid := "user-1"
	pickRepo := &mockPickingTaskRepoCtrl{
		tasks: []responses.PickingTaskView{
			{ID: "pt-1", TaskID: "T1", AssignedTo: &uid, Status: "in_progress"},
			{ID: "pt-2", TaskID: "T2", AssignedTo: nil, Status: "in_progress"},
		},
	}
	pickSvc := services.NewPickingTaskService(pickRepo)
	cfg := configuration.Config{JWTSecret: testJWTSecret}
	ctrl := NewMobileController(pickSvc, nil, nil, nil, nil, nil, cfg)

	w := performRequestWithHeader(ctrl.ListPickingTasks, "GET", "/api/mobile/picking-tasks?assigned_to_me=true", nil, nil, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)

	var env responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	arr, ok := env.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 1)
}

// ─── /api/mobile/picking-tasks/:id/complete-line ─────────────────────────────

func TestMobileController_CompletePickingLine_BadJSON(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		nil, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMobileController_CompletePickingLine_MissingLocation(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 1}
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMobileController_CompletePickingLine_Unauthorized(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 1, LocationScanned: "LOC-A"}
	w := performRequest(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "pt-1"}})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMobileController_CompletePickingLine_Success(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 1, LocationScanned: "LOC-A"}
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

// W0.7 N1-2: backend now validates picked_qty <= expected_qty * 1.05 server-side.
// Sample task line is expected=10, so picked=11 (>10.5) MUST 400.
func TestMobile_CompletePickingLine_RejectsBeyondTolerance(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 11, LocationScanned: "LOC-A"}
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// W0.7 N1-2: at the tolerance boundary (10 * 1.05 = 10.5) the request passes;
// guards the boundary check against off-by-one regressions.
func TestMobile_CompletePickingLine_AcceptsAtToleranceBoundary(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 10.5, LocationScanned: "LOC-A"}
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

// W0.7 N1-2: zero / negative picked_qty must 400 with the dedicated error
// rather than silently passing through to the service layer.
func TestMobile_CompletePickingLine_RejectsZeroQty(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 0, LocationScanned: "LOC-A"}
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// W0.7 N1-2: an unknown line_id must 400 — backend will not silently fall back
// to the (sku, lot, serial) tuple when an explicit identifier is supplied
// because that would mask a real client/server contract drift.
func TestMobile_CompletePickingLine_UnknownLineID_400(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{
		LineID:          "deadbeefcafe",
		SKU:             "SKU-1",
		PickedQty:       1,
		LocationScanned: "LOC-A",
	}
	w := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// W0.7 N1-1: GetPickingTask now returns MobilePickingTaskDetailDto with flat
// lines. Each line MUST carry a non-empty line_id and (when allocations
// existed on the source item) a non-empty location.
func TestMobile_GetPickingTask_ReturnsDetailDto_WithLineIDAndLocation(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequestWithHeader(ctrl.GetPickingTask, "GET", "/api/mobile/picking-tasks/pt-1",
		nil, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)

	var env responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))

	dataMap, ok := env.Data.(map[string]interface{})
	require.True(t, ok, "data should be an object")
	assert.Equal(t, "pt-1", dataMap["id"])
	assert.Equal(t, "T1", dataMap["task_id"])
	linesAny, ok := dataMap["lines"].([]interface{})
	require.True(t, ok, "lines should be an array")
	require.Len(t, linesAny, 1, "sample task has one item")

	line, ok := linesAny[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "SKU-1", line["sku"])
	assert.Equal(t, "LOC-A", line["location"], "location must resolve from allocations[0].location")
	if lineID, _ := line["line_id"].(string); lineID == "" {
		t.Fatalf("line_id must be non-empty (got empty)")
	}
	assert.Equal(t, float64(10), line["expected_qty"])
	assert.Equal(t, "pending", line["status"])
}

// W0.7 N1-1: line_id round-trips deterministically — the LineID emitted by
// GET MUST match what CompletePickingLine recomputes from the same task,
// which is what makes the contract usable without persistence.
func TestMobile_GetPickingTask_LineID_RoundTripsToCompleteLine(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequestWithHeader(ctrl.GetPickingTask, "GET", "/api/mobile/picking-tasks/pt-1",
		nil, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)
	var env responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	dataMap := env.Data.(map[string]interface{})
	line := dataMap["lines"].([]interface{})[0].(map[string]interface{})
	lineID := line["line_id"].(string)
	require.NotEmpty(t, lineID)

	body := responses.MobileCompleteLineRequest{
		LineID:          lineID,
		SKU:             "SKU-1",
		PickedQty:       5,
		LocationScanned: "LOC-A",
	}
	w2 := performRequestWithHeader(ctrl.CompletePickingLine, "PATCH", "/api/mobile/picking-tasks/pt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w2.Code)
}

// W0.7 Fix D: CompletePickingTask must accept an empty body (post-W0.6 the
// service does not consume location_scanned anymore).
func TestMobile_CompletePickingTask_AcceptsEmptyBody(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequestWithHeader(ctrl.CompletePickingTask, "PATCH", "/api/mobile/picking-tasks/pt-1/complete",
		nil, gin.Params{{Key: "id", Value: "pt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── /api/mobile/counts/:id/scan-line ────────────────────────────────────────

// Reuses mockCountsRepo defined in services_test.go via local copies — but tests are at controller level,
// so we drive via real service backed by an inline mock repo (simpler than building DI mock infra).

type ctrlCountsRepoStub struct {
	count       *database.InventoryCount
	expectedQty float64
	addedLines  []database.InventoryCountLine
}

func (m *ctrlCountsRepoStub) List(status, locationID string) ([]database.InventoryCount, *responses.InternalResponse) {
	if m.count == nil {
		return nil, nil
	}
	return []database.InventoryCount{*m.count}, nil
}
func (m *ctrlCountsRepoStub) GetByID(id string) (*database.InventoryCount, *responses.InternalResponse) {
	return m.count, nil
}
func (m *ctrlCountsRepoStub) GetDetail(id string) (*responses.InventoryCountDetail, *responses.InternalResponse) {
	if m.count == nil {
		return nil, &responses.InternalResponse{Message: "nf", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return &responses.InventoryCountDetail{Count: *m.count}, nil
}
func (m *ctrlCountsRepoStub) Create(userID string, req *requests.CreateInventoryCount) (*database.InventoryCount, *responses.InternalResponse) {
	c := &database.InventoryCount{ID: "c1", Code: req.Code, Name: req.Name, Status: "draft"}
	m.count = c
	return c, nil
}
func (m *ctrlCountsRepoStub) UpdateStatus(id, status string) *responses.InternalResponse { return nil }
func (m *ctrlCountsRepoStub) MarkStarted(id string) *responses.InternalResponse {
	if m.count != nil {
		m.count.Status = "in_progress"
	}
	return nil
}
func (m *ctrlCountsRepoStub) MarkCancelled(id string) *responses.InternalResponse { return nil }
func (m *ctrlCountsRepoStub) MarkSubmitted(id, submittedBy, adjustmentID string) *responses.InternalResponse {
	if m.count != nil {
		m.count.Status = "submitted"
	}
	return nil
}
func (m *ctrlCountsRepoStub) SubmitWithAdjustments(countID, userID string, creator ports.InventoryAdjustmentsCreator) *responses.InternalResponse {
	if m.count != nil {
		m.count.Status = "submitted"
	}
	return nil
}
func (m *ctrlCountsRepoStub) ListLines(countID string) ([]database.InventoryCountLine, *responses.InternalResponse) {
	return m.addedLines, nil
}
func (m *ctrlCountsRepoStub) AddLine(line *database.InventoryCountLine) *responses.InternalResponse {
	if line.ID == "" {
		line.ID = "line-1"
	}
	m.addedLines = append(m.addedLines, *line)
	return nil
}
func (m *ctrlCountsRepoStub) ListLocations(countID string) ([]database.InventoryCountLocation, *responses.InternalResponse) {
	return nil, nil
}
func (m *ctrlCountsRepoStub) ResolveSKUByBarcode(barcode string) (string, *responses.InternalResponse) {
	return "", nil
}
func (m *ctrlCountsRepoStub) GetExpectedQty(sku, loc, lot string) (float64, *responses.InternalResponse) {
	return m.expectedQty, nil
}
func (m *ctrlCountsRepoStub) GetLocationCodeByID(id string) (string, *responses.InternalResponse) {
	return "LOC-A", nil
}

func newTestCountsController(repo *ctrlCountsRepoStub) *InventoryCountsController {
	svc := services.NewInventoryCountsService(repo, nil)
	return NewInventoryCountsController(*svc, testJWTSecret)
}

func TestInventoryCountsController_ScanLine_Success(t *testing.T) {
	repo := &ctrlCountsRepoStub{
		count:       &database.InventoryCount{ID: "c1", Status: "in_progress"},
		expectedQty: 5,
	}
	ctrl := newTestCountsController(repo)
	body := requests.ScanCountLine{LocationID: "loc-1", SKU: "SKU-1", ScannedQty: 7}
	w := performRequestWithHeader(ctrl.ScanLine, "POST", "/api/mobile/counts/c1/scan-line", body,
		gin.Params{{Key: "id", Value: "c1"}}, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, repo.addedLines, 1)
	assert.Equal(t, 2.0, repo.addedLines[0].VarianceQty)
}

func TestInventoryCountsController_ScanLine_Unauthorized(t *testing.T) {
	repo := &ctrlCountsRepoStub{count: &database.InventoryCount{ID: "c1", Status: "in_progress"}}
	ctrl := newTestCountsController(repo)
	body := requests.ScanCountLine{LocationID: "loc-1", SKU: "SKU-1", ScannedQty: 1}
	w := performRequest(ctrl.ScanLine, "POST", "/api/mobile/counts/c1/scan-line", body,
		gin.Params{{Key: "id", Value: "c1"}})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInventoryCountsController_Submit_Success(t *testing.T) {
	repo := &ctrlCountsRepoStub{count: &database.InventoryCount{ID: "c1", Status: "in_progress"}}
	ctrl := newTestCountsController(repo)
	w := performRequestWithHeader(ctrl.Submit, "POST", "/api/mobile/counts/c1/submit", nil,
		gin.Params{{Key: "id", Value: "c1"}}, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryCountsController_Submit_Unauthorized(t *testing.T) {
	repo := &ctrlCountsRepoStub{count: &database.InventoryCount{ID: "c1", Status: "in_progress"}}
	ctrl := newTestCountsController(repo)
	w := performRequest(ctrl.Submit, "POST", "/api/mobile/counts/c1/submit", nil,
		gin.Params{{Key: "id", Value: "c1"}})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInventoryCountsController_GetDetail_Success(t *testing.T) {
	repo := &ctrlCountsRepoStub{count: &database.InventoryCount{ID: "c1", Code: "CC-001", Status: "draft"}}
	ctrl := newTestCountsController(repo)
	w := performRequest(ctrl.GetDetail, "GET", "/api/mobile/counts/c1", nil,
		gin.Params{{Key: "id", Value: "c1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInventoryCountsController_GetDetail_NotFound(t *testing.T) {
	repo := &ctrlCountsRepoStub{count: nil}
	ctrl := newTestCountsController(repo)
	w := performRequest(ctrl.GetDetail, "GET", "/api/mobile/counts/missing", nil,
		gin.Params{{Key: "id", Value: "missing"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}
