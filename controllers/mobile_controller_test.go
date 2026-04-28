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
	"github.com/eflowcr/eSTOCK_backend/tools"
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

// buildReceivingItemsJSON returns the jsonb encoding of one or more receiving
// items, panicking on marshaling errors (test code only).
func buildReceivingItemsJSON(items ...database.ReceivingTaskItem) []byte {
	b, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}
	return b
}

// sampleReceivingTaskWithItem returns a ReceivingTask with a single item
// (SKU-1, expected 10, at LOC-A). Used by the W7 N1-B receiving contract tests
// — same fixture pattern as sampleTaskWithItem (picking).
func sampleReceivingTaskWithItem() *database.ReceivingTask {
	items := buildReceivingItemsJSON(database.ReceivingTaskItem{
		SKU:              "SKU-1",
		ExpectedQuantity: 10,
		Location:         "LOC-A",
	})
	return &database.ReceivingTask{
		ID:            "rt-1",
		TaskID:        "RT1",
		InboundNumber: "IB-1",
		Status:        "in_progress",
		Items:         items,
	}
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
		byID:  map[string]*database.ReceivingTask{"rt-1": sampleReceivingTaskWithItem()},
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
func (m *ctrlCountsRepoStub) GetLocationIDByCode(code string) (string, *responses.InternalResponse) {
	// W7 N1-A: stub maps known codes to deterministic UUIDs so the controller
	// path that calls ResolveLocationIDByCode is observable in tests. The
	// "BIN-NOEXIST" sentinel exercises the not-found error path. Any other
	// non-UUID code (e.g. legacy short ids like "loc-1" used by pre-W7 tests)
	// resolves echo-style to keep back-compat tests green.
	switch code {
	case "BIN-A1":
		return "11111111-1111-1111-1111-111111111111", nil
	case "BIN-NOEXIST":
		return "", &responses.InternalResponse{
			Message:    "Ubicación no encontrada con código " + code,
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		}
	default:
		return code, nil
	}
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

// ─── W7 N1-A: counts location code → UUID resolver ───────────────────────────

// TestScanCountLine_ResolvesLocationCode: mobile sends a printed location code
// ("BIN-A1") in the location_id field. The controller detects the non-UUID
// shape, resolves it via GetLocationIDByCode, and ScanLine receives the
// resolved UUID — which is what the inventory_count_lines.location_id FK
// expects. Pre-W7 this would have failed with FK violation in production.
func TestScanCountLine_ResolvesLocationCode(t *testing.T) {
	repo := &ctrlCountsRepoStub{
		count:       &database.InventoryCount{ID: "c1", Status: "in_progress"},
		expectedQty: 5,
	}
	ctrl := newTestCountsController(repo)
	body := requests.ScanCountLine{LocationID: "BIN-A1", SKU: "SKU-1", ScannedQty: 7}
	w := performRequestWithHeader(ctrl.ScanLine, "POST", "/api/mobile/counts/c1/scan-line", body,
		gin.Params{{Key: "id", Value: "c1"}}, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, repo.addedLines, 1)
	// The resolver mapped BIN-A1 → 11111111-1111-1111-1111-111111111111. The
	// persisted line MUST carry the resolved UUID, not the scanned code.
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", repo.addedLines[0].LocationID)
}

// TestScanCountLine_RejectsUnknownLocationCode: a code that doesn't resolve
// returns 404 (resolver returns NotFound, controller forwards via writeErrorResponse).
func TestScanCountLine_RejectsUnknownLocationCode(t *testing.T) {
	repo := &ctrlCountsRepoStub{
		count: &database.InventoryCount{ID: "c1", Status: "in_progress"},
	}
	ctrl := newTestCountsController(repo)
	body := requests.ScanCountLine{LocationID: "BIN-NOEXIST", SKU: "SKU-1", ScannedQty: 1}
	w := performRequestWithHeader(ctrl.ScanLine, "POST", "/api/mobile/counts/c1/scan-line", body,
		gin.Params{{Key: "id", Value: "c1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, repo.addedLines, "line must NOT be persisted when location lookup fails")
}

// TestScanCountLine_AcceptsUUIDDirectly: when the body carries a real UUID,
// the controller bypasses the resolver and the value flows straight through
// to AddLine. Guarantees back-compat with admin/web clients sending the UUID.
func TestScanCountLine_AcceptsUUIDDirectly(t *testing.T) {
	uuid := "22222222-2222-2222-2222-222222222222"
	repo := &ctrlCountsRepoStub{
		count:       &database.InventoryCount{ID: "c1", Status: "in_progress"},
		expectedQty: 3,
	}
	ctrl := newTestCountsController(repo)
	body := requests.ScanCountLine{LocationID: uuid, SKU: "SKU-1", ScannedQty: 4}
	w := performRequestWithHeader(ctrl.ScanLine, "POST", "/api/mobile/counts/c1/scan-line", body,
		gin.Params{{Key: "id", Value: "c1"}}, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, repo.addedLines, 1)
	assert.Equal(t, uuid, repo.addedLines[0].LocationID, "UUID must pass through unchanged")
}

// ─── W7 N1-B: receiving contract parity (mirrors W0.7 picking tests) ─────────

// TestMobile_GetReceivingTask_ReturnsDetailDto_WithLineIDAndLocation: the
// receiving GET endpoint must return MobileReceivingTaskDetailDto (flat shape)
// with a non-empty line_id and location resolved per line — same contract as
// picking after W0.7.
func TestMobile_GetReceivingTask_ReturnsDetailDto_WithLineIDAndLocation(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequestWithHeader(ctrl.GetReceivingTask, "GET", "/api/mobile/receiving-tasks/rt-1",
		nil, gin.Params{{Key: "id", Value: "rt-1"}}, map[string]string{"Authorization": makeTestToken()})
	require.Equal(t, http.StatusOK, w.Code)

	var env responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	dataMap, ok := env.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "rt-1", dataMap["id"])
	linesAny, ok := dataMap["lines"].([]interface{})
	require.True(t, ok, "lines must be an array")
	require.Len(t, linesAny, 1)

	line, ok := linesAny[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "SKU-1", line["sku"])
	assert.Equal(t, "LOC-A", line["location"], "location must come from item.Location")
	if lineID, _ := line["line_id"].(string); lineID == "" {
		t.Fatalf("line_id must be non-empty (got empty)")
	}
	assert.Equal(t, float64(10), line["expected_qty"])
	assert.Equal(t, "pending", line["status"])
}

// TestMobile_LineID_RoundTripsToCompleteLine: the LineID emitted by the GET
// receiving detail endpoint MUST match what CompleteReceivingLine recomputes
// from the same task — making the determinism contract usable without
// persistence (mirrors picking W0.7).
func TestMobile_LineID_RoundTripsToCompleteLine(t *testing.T) {
	ctrl := newTestMobileController()
	w := performRequestWithHeader(ctrl.GetReceivingTask, "GET", "/api/mobile/receiving-tasks/rt-1",
		nil, gin.Params{{Key: "id", Value: "rt-1"}}, map[string]string{"Authorization": makeTestToken()})
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
	w2 := performRequestWithHeader(ctrl.CompleteReceivingLine, "PATCH", "/api/mobile/receiving-tasks/rt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "rt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w2.Code)
}

// TestMobile_CompleteReceivingLine_RejectsBeyondTolerance: expected=10, picked=11
// (>10.5) MUST 400. Pre-W7 the controller synthesized expected=picked=11 so any
// over-receive passed silently.
func TestMobile_CompleteReceivingLine_RejectsBeyondTolerance(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 11, LocationScanned: "LOC-A"}
	w := performRequestWithHeader(ctrl.CompleteReceivingLine, "PATCH", "/api/mobile/receiving-tasks/rt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "rt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestMobile_CompleteReceivingLine_AcceptsAtToleranceBoundary: at the boundary
// (10 * 1.05 = 10.5) the request passes — guards against off-by-one.
func TestMobile_CompleteReceivingLine_AcceptsAtToleranceBoundary(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 10.5, LocationScanned: "LOC-A"}
	w := performRequestWithHeader(ctrl.CompleteReceivingLine, "PATCH", "/api/mobile/receiving-tasks/rt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "rt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestMobile_CompleteReceivingLine_RejectsZeroQty: zero/negative received_qty
// must 400 with the dedicated error rather than passing through to the service.
func TestMobile_CompleteReceivingLine_RejectsZeroQty(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{SKU: "SKU-1", PickedQty: 0, LocationScanned: "LOC-A"}
	w := performRequestWithHeader(ctrl.CompleteReceivingLine, "PATCH", "/api/mobile/receiving-tasks/rt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "rt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestMobile_CompleteReceivingLine_UnknownLineID_400: an explicit LineID that
// doesn't resolve must 400 — backend will not silently fall back to (sku,lot,
// serial) tuple when a deterministic identifier is supplied (would mask
// client/server contract drift).
func TestMobile_CompleteReceivingLine_UnknownLineID_400(t *testing.T) {
	ctrl := newTestMobileController()
	body := responses.MobileCompleteLineRequest{
		LineID:          "deadbeefcafe",
		SKU:             "SKU-1",
		PickedQty:       1,
		LocationScanned: "LOC-A",
	}
	w := performRequestWithHeader(ctrl.CompleteReceivingLine, "PATCH", "/api/mobile/receiving-tasks/rt-1/complete-line",
		body, gin.Params{{Key: "id", Value: "rt-1"}}, map[string]string{"Authorization": makeTestToken()})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── W7 N2-1: operator role forces assigned_to_me=true ───────────────────────

// makeOperatorTestToken returns a JWT with role="operator" so the mobile list
// handlers' role override path is exercised.
func makeOperatorTestToken() string {
	token, _ := tools.GenerateToken(testJWTSecret, "user-1", "testuser", "test@test.com", "operator")
	return "Bearer " + token
}

// TestListPickingTasks_OperatorRoleForcesAssignedToMe: an operator hitting the
// list endpoint without ?assigned_to_me=true MUST be implicitly filtered to
// their own tasks. Without the override, operator A could observe operator B's
// task assignments (W7 N2-1 data leak fix).
func TestListPickingTasks_OperatorRoleForcesAssignedToMe(t *testing.T) {
	uid := "user-1"
	other := "operator-2"
	pickRepo := &mockPickingTaskRepoCtrl{
		tasks: []responses.PickingTaskView{
			{ID: "pt-1", TaskID: "T1", AssignedTo: &uid, Status: "in_progress"},
			{ID: "pt-2", TaskID: "T2", AssignedTo: &other, Status: "in_progress"},
			{ID: "pt-3", TaskID: "T3", AssignedTo: nil, Status: "in_progress"},
		},
	}
	pickSvc := services.NewPickingTaskService(pickRepo)
	cfg := configuration.Config{JWTSecret: testJWTSecret}
	ctrl := NewMobileController(pickSvc, nil, nil, nil, nil, nil, cfg)

	// Operator role + NO assigned_to_me flag → must be forced.
	w := performRequestWithHeader(ctrl.ListPickingTasks, "GET", "/api/mobile/picking-tasks",
		nil, nil, map[string]string{"Authorization": makeOperatorTestToken()})
	require.Equal(t, http.StatusOK, w.Code)

	var env responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	arr, ok := env.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, arr, 1, "operator must only see tasks assigned to themselves")
	first := arr[0].(map[string]interface{})
	assert.Equal(t, "pt-1", first["id"])
}
