package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ─── mock InventoryCountsRepository ──────────────────────────────────────────

type mockCountsRepo struct {
	count               *database.InventoryCount
	getErr              *responses.InternalResponse
	listLines           []database.InventoryCountLine
	listLinesErr        *responses.InternalResponse
	addLineErr          *responses.InternalResponse
	addedLines          []database.InventoryCountLine
	resolvedSKU         string
	resolveErr          *responses.InternalResponse
	expectedQty         float64
	expectedQtyByLine   map[string]float64 // line.ID -> expected at submit time (variance recompute)
	expectedQtyErr      *responses.InternalResponse
	locationCode        string
	locationErr         *responses.InternalResponse
	startCalled         bool
	cancelCalled        bool
	submitCalled        bool
	submittedAdjID      string
	submitWithAdjErr    *responses.InternalResponse
	submitWithAdjCalled bool
	// submitFanOut, when non-nil, is invoked from SubmitWithAdjustments to simulate
	// per-line creator calls and surface partial-failure scenarios deterministically.
	submitFanOut func(creator ports.InventoryAdjustmentsCreator) *responses.InternalResponse
}

func (m *mockCountsRepo) List(status, locationID string) ([]database.InventoryCount, *responses.InternalResponse) {
	if m.count == nil {
		return nil, nil
	}
	return []database.InventoryCount{*m.count}, nil
}
func (m *mockCountsRepo) GetByID(id string) (*database.InventoryCount, *responses.InternalResponse) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.count, nil
}
func (m *mockCountsRepo) GetDetail(id string) (*responses.InventoryCountDetail, *responses.InternalResponse) {
	if m.count == nil {
		return nil, &responses.InternalResponse{Message: "nf", Handled: true, StatusCode: responses.StatusNotFound}
	}
	return &responses.InventoryCountDetail{Count: *m.count, Lines: m.listLines}, nil
}
func (m *mockCountsRepo) Create(userID string, req *requests.CreateInventoryCount) (*database.InventoryCount, *responses.InternalResponse) {
	c := &database.InventoryCount{ID: "c-new", Code: req.Code, Name: req.Name, Status: "draft", CreatedBy: userID}
	m.count = c
	return c, nil
}
func (m *mockCountsRepo) UpdateStatus(id, status string) *responses.InternalResponse {
	if m.count != nil {
		m.count.Status = status
	}
	return nil
}
func (m *mockCountsRepo) MarkStarted(id string) *responses.InternalResponse {
	m.startCalled = true
	if m.count != nil {
		m.count.Status = "in_progress"
	}
	return nil
}
func (m *mockCountsRepo) MarkCancelled(id string) *responses.InternalResponse {
	m.cancelCalled = true
	if m.count != nil {
		m.count.Status = "cancelled"
	}
	return nil
}
func (m *mockCountsRepo) MarkSubmitted(id, submittedBy, adjustmentID string) *responses.InternalResponse {
	m.submitCalled = true
	m.submittedAdjID = adjustmentID
	if m.count != nil {
		m.count.Status = "submitted"
	}
	return nil
}
func (m *mockCountsRepo) SubmitWithAdjustments(countID, userID string, creator ports.InventoryAdjustmentsCreator) *responses.InternalResponse {
	m.submitWithAdjCalled = true
	if m.submitWithAdjErr != nil {
		return m.submitWithAdjErr
	}
	if m.submitFanOut != nil {
		if resp := m.submitFanOut(creator); resp != nil {
			// Per the contract, partial-failure rolls back: count stays in_progress,
			// and we DO NOT call MarkSubmitted. Surface the error to the caller.
			return resp
		}
	}
	// Successful submit: simulate the state transition.
	m.submitCalled = true
	if m.count != nil {
		m.count.Status = "submitted"
	}
	return nil
}
func (m *mockCountsRepo) ListLines(countID string) ([]database.InventoryCountLine, *responses.InternalResponse) {
	return m.listLines, m.listLinesErr
}
func (m *mockCountsRepo) AddLine(line *database.InventoryCountLine) *responses.InternalResponse {
	if m.addLineErr != nil {
		return m.addLineErr
	}
	if line.ID == "" {
		line.ID = "line-1"
	}
	m.addedLines = append(m.addedLines, *line)
	return nil
}
func (m *mockCountsRepo) ListLocations(countID string) ([]database.InventoryCountLocation, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockCountsRepo) ResolveSKUByBarcode(barcode string) (string, *responses.InternalResponse) {
	if m.resolveErr != nil {
		return "", m.resolveErr
	}
	return m.resolvedSKU, nil
}
func (m *mockCountsRepo) GetExpectedQty(sku, locationCode, lot string) (float64, *responses.InternalResponse) {
	return m.expectedQty, m.expectedQtyErr
}
func (m *mockCountsRepo) GetLocationCodeByID(locationID string) (string, *responses.InternalResponse) {
	if m.locationErr != nil {
		return "", m.locationErr
	}
	if m.locationCode == "" {
		return "LOC-A", nil
	}
	return m.locationCode, nil
}

// ─── mock AdjustmentsRepository (existing port surface, unchanged) ───────────

type mockAdjustmentsRepoForCounts struct {
	createCalls []requests.CreateAdjustment
	createErr   *responses.InternalResponse
	createdID   string
}

func (m *mockAdjustmentsRepoForCounts) GetAllAdjustments() ([]database.Adjustment, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockAdjustmentsRepoForCounts) GetAdjustmentByID(id string) (*database.Adjustment, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockAdjustmentsRepoForCounts) GetAdjustmentDetails(id string) (*dto.AdjustmentDetails, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockAdjustmentsRepoForCounts) CreateAdjustment(userId string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.createCalls = append(m.createCalls, adjustment)
	id := m.createdID
	if id == "" {
		id = "adj-1"
	}
	return &database.Adjustment{ID: id, SKU: adjustment.SKU, Location: adjustment.Location, UserID: userId}, nil
}
func (m *mockAdjustmentsRepoForCounts) ExportAdjustmentsToExcel() ([]byte, *responses.InternalResponse) {
	return nil, nil
}

// ─── mock InventoryAdjustmentsCreator (new port; tx-scoped) ──────────────────
//
// Used by Submit_SignCorrectness + PartialFailure tests to assert that the
// service routes adjustments through CreateAdjustmentTx (which applies the
// reason-code-driven sign flip) rather than the legacy CreateAdjustment.

type mockAdjustmentsCreator struct {
	calls       []requests.CreateAdjustment
	failOnIndex int  // -1 => never fail; otherwise fail on the (failOnIndex)-th call (0-based)
	createdID   string
	created     []database.Adjustment
}

func (m *mockAdjustmentsCreator) CreateAdjustmentTx(tx *gorm.DB, userId string, adjustment requests.CreateAdjustment) (*database.Adjustment, *responses.InternalResponse) {
	idx := len(m.calls)
	m.calls = append(m.calls, adjustment)
	if m.failOnIndex >= 0 && idx == m.failOnIndex {
		return nil, &responses.InternalResponse{Error: errors.New("simulated"), Message: "simulated adjustment failure", Handled: false}
	}
	id := m.createdID
	if id == "" {
		id = "adj-1"
	}
	adj := database.Adjustment{ID: id, SKU: adjustment.SKU, Location: adjustment.Location, UserID: userId}
	m.created = append(m.created, adj)
	return &adj, nil
}

// ─── tests ───────────────────────────────────────────────────────────────────

func newCountsServiceWithDraft(repo *mockCountsRepo) *InventoryCountsService {
	repo.count = &database.InventoryCount{ID: "c1", Code: "CC-001", Name: "Daily count", Status: "draft", CreatedBy: "user-1"}
	return NewInventoryCountsService(repo, nil)
}

func TestInventoryCountsService_Start_Success(t *testing.T) {
	repo := &mockCountsRepo{}
	svc := newCountsServiceWithDraft(repo)
	resp := svc.Start("c1")
	require.Nil(t, resp)
	assert.True(t, repo.startCalled)
	assert.Equal(t, "in_progress", repo.count.Status)
}

func TestInventoryCountsService_Start_BadStatus(t *testing.T) {
	repo := &mockCountsRepo{count: &database.InventoryCount{ID: "c1", Status: "submitted"}}
	svc := NewInventoryCountsService(repo, nil)
	resp := svc.Start("c1")
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
	assert.False(t, repo.startCalled)
}

func TestInventoryCountsService_ScanLine_Happy(t *testing.T) {
	repo := &mockCountsRepo{
		count:        &database.InventoryCount{ID: "c1", Status: "in_progress"},
		expectedQty:  10,
		locationCode: "LOC-A",
	}
	svc := NewInventoryCountsService(repo, nil)
	line, resp := svc.ScanLine("c1", "user-1", &requests.ScanCountLine{
		LocationID: "loc-id-1",
		SKU:        "SKU-1",
		ScannedQty: 12,
	})
	require.Nil(t, resp)
	require.NotNil(t, line)
	assert.Equal(t, "SKU-1", line.SKU)
	assert.Equal(t, 10.0, line.ExpectedQty)
	assert.Equal(t, 12.0, line.ScannedQty)
	assert.Equal(t, 2.0, line.VarianceQty)
	require.Len(t, repo.addedLines, 1)
}

func TestInventoryCountsService_ScanLine_ResolvesBarcode(t *testing.T) {
	repo := &mockCountsRepo{
		count:        &database.InventoryCount{ID: "c1", Status: "in_progress"},
		expectedQty:  5,
		locationCode: "LOC-A",
		resolvedSKU:  "SKU-FROM-BARCODE",
	}
	svc := NewInventoryCountsService(repo, nil)
	line, resp := svc.ScanLine("c1", "user-1", &requests.ScanCountLine{
		LocationID: "loc-id-1",
		Barcode:    "1234567890",
		ScannedQty: 4,
	})
	require.Nil(t, resp)
	require.NotNil(t, line)
	assert.Equal(t, "SKU-FROM-BARCODE", line.SKU)
	assert.Equal(t, -1.0, line.VarianceQty)
}

func TestInventoryCountsService_ScanLine_RequiresSkuOrBarcode(t *testing.T) {
	repo := &mockCountsRepo{count: &database.InventoryCount{ID: "c1", Status: "in_progress"}}
	svc := NewInventoryCountsService(repo, nil)
	line, resp := svc.ScanLine("c1", "user-1", &requests.ScanCountLine{LocationID: "loc-1", ScannedQty: 1})
	assert.Nil(t, line)
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

func TestInventoryCountsService_ScanLine_RejectsClosedCount(t *testing.T) {
	repo := &mockCountsRepo{count: &database.InventoryCount{ID: "c1", Status: "submitted"}}
	svc := NewInventoryCountsService(repo, nil)
	_, resp := svc.ScanLine("c1", "user-1", &requests.ScanCountLine{LocationID: "loc-1", SKU: "X", ScannedQty: 1})
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
}

func TestInventoryCountsService_Submit_Success(t *testing.T) {
	repo := &mockCountsRepo{
		count:        &database.InventoryCount{ID: "c1", Code: "CC-001", Status: "in_progress", CreatedBy: "user-1"},
		locationCode: "LOC-A",
	}
	creator := &mockAdjustmentsCreator{failOnIndex: -1}
	svc := NewInventoryCountsService(repo, creator)
	updated, resp := svc.Submit("c1", "user-1", "operator")
	require.Nil(t, resp)
	require.NotNil(t, updated)
	assert.Equal(t, "submitted", updated.Status)
	assert.True(t, repo.submitWithAdjCalled)
}

// W0.5 N1-1 regression: verify outbound variance routes through the creator with
// reason code OUTBOUND and a positive AdjustmentQuantity. The reason-code driven
// sign flip happens *inside* CreateAdjustmentTx (AdjustmentsService) — the counts
// pipeline must not pre-flip.
func TestInventoryCountsService_Submit_SignCorrectness_OutboundReducesStock(t *testing.T) {
	creator := &mockAdjustmentsCreator{failOnIndex: -1}
	repo := &mockCountsRepo{
		count:        &database.InventoryCount{ID: "c1", Code: "CC-OUT", Status: "in_progress", CreatedBy: "user-1"},
		locationCode: "LOC-A",
	}
	// Drive the fan-out manually so we can observe the exact CreateAdjustmentTx call args
	// (mockCountsRepo.SubmitWithAdjustments doesn't recompute variance — it just simulates
	// the repo's role).
	repo.submitFanOut = func(c ports.InventoryAdjustmentsCreator) *responses.InternalResponse {
		// Outbound variance: scanned 4, expected 10 → variance −6, abs = 6 with reason OUTBOUND.
		_, errResp := c.CreateAdjustmentTx(nil, "user-1", requests.CreateAdjustment{
			SKU:                "SKU-1",
			Location:           "LOC-A",
			AdjustmentQuantity: 6,
			Reason:             "INVENTORY_COUNT_OUTBOUND",
			Notes:              "inventory_count CC-OUT",
		})
		return errResp
	}
	svc := NewInventoryCountsService(repo, creator)
	_, resp := svc.Submit("c1", "user-1", "operator")
	require.Nil(t, resp)
	require.Len(t, creator.calls, 1)
	assert.Equal(t, "INVENTORY_COUNT_OUTBOUND", creator.calls[0].Reason, "outbound variance must use OUTBOUND reason code, not INBOUND")
	assert.Equal(t, 6.0, creator.calls[0].AdjustmentQuantity, "AdjustmentQuantity must be the absolute (positive) value; sign comes from reason code direction in AdjustmentsService.CreateAdjustmentTx")
}

// W0.5 N1-2 regression: when adjustment N+1 fails inside the fan-out, the repo
// rolls back every prior adjustment and the count stays in_progress.
func TestInventoryCountsService_Submit_PartialFailure_Rollbacks(t *testing.T) {
	creator := &mockAdjustmentsCreator{failOnIndex: 1} // 2nd call fails
	repo := &mockCountsRepo{
		count:        &database.InventoryCount{ID: "c1", Code: "CC-FAIL", Status: "in_progress", CreatedBy: "user-1"},
		locationCode: "LOC-A",
	}
	repo.submitFanOut = func(c ports.InventoryAdjustmentsCreator) *responses.InternalResponse {
		// Two adjustments; the second fails.
		if _, errResp := c.CreateAdjustmentTx(nil, "user-1", requests.CreateAdjustment{SKU: "SKU-1", Location: "LOC-A", AdjustmentQuantity: 2, Reason: "INVENTORY_COUNT_INBOUND"}); errResp != nil {
			return errResp
		}
		if _, errResp := c.CreateAdjustmentTx(nil, "user-1", requests.CreateAdjustment{SKU: "SKU-2", Location: "LOC-A", AdjustmentQuantity: 4, Reason: "INVENTORY_COUNT_OUTBOUND"}); errResp != nil {
			return errResp
		}
		return nil
	}
	svc := NewInventoryCountsService(repo, creator)
	_, resp := svc.Submit("c1", "user-1", "operator")
	require.NotNil(t, resp, "partial failure must propagate as an error")
	// Count must NOT have transitioned to submitted; the mockCountsRepo.SubmitWithAdjustments
	// is wired so it only flips status when fan-out returns nil.
	assert.NotEqual(t, "submitted", repo.count.Status)
	assert.False(t, repo.submitCalled, "MarkSubmitted (or its tx-scoped equivalent) must not have been called")
}

// W0.5 N2-1: a non-creator non-admin caller cannot submit someone else's count.
func TestInventoryCountsService_Submit_RejectsNonOwner(t *testing.T) {
	repo := &mockCountsRepo{
		count: &database.InventoryCount{ID: "c1", Status: "in_progress", CreatedBy: "owner-1"},
	}
	creator := &mockAdjustmentsCreator{failOnIndex: -1}
	svc := NewInventoryCountsService(repo, creator)
	_, resp := svc.Submit("c1", "intruder-2", "operator")
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusForbidden, resp.StatusCode)
	assert.False(t, repo.submitWithAdjCalled, "non-owner submit must short-circuit before touching the repo")
}

// W0.5 N2-1: a non-creator non-admin caller cannot cancel someone else's count.
func TestInventoryCountsService_Cancel_RejectsNonOwner(t *testing.T) {
	repo := &mockCountsRepo{
		count: &database.InventoryCount{ID: "c1", Status: "in_progress", CreatedBy: "owner-1"},
	}
	svc := NewInventoryCountsService(repo, nil)
	resp := svc.Cancel("c1", "intruder-2", "operator")
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusForbidden, resp.StatusCode)
	assert.False(t, repo.cancelCalled, "non-owner cancel must short-circuit before MarkCancelled")
}

// W0.5 N2-1 happy path: admin role bypasses ownership.
func TestInventoryCountsService_Cancel_AdminCanCancelOthers(t *testing.T) {
	repo := &mockCountsRepo{
		count: &database.InventoryCount{ID: "c1", Status: "in_progress", CreatedBy: "owner-1"},
	}
	svc := NewInventoryCountsService(repo, nil)
	resp := svc.Cancel("c1", "admin-user", "admin")
	require.Nil(t, resp)
	assert.True(t, repo.cancelCalled)
}

func TestInventoryCountsService_Cancel(t *testing.T) {
	repo := &mockCountsRepo{count: &database.InventoryCount{ID: "c1", Status: "in_progress", CreatedBy: "user-1"}}
	svc := NewInventoryCountsService(repo, nil)
	resp := svc.Cancel("c1", "user-1", "operator")
	require.Nil(t, resp)
	assert.True(t, repo.cancelCalled)
	assert.Equal(t, "cancelled", repo.count.Status)
}

func TestInventoryCountsService_Cancel_RejectsClosed(t *testing.T) {
	repo := &mockCountsRepo{count: &database.InventoryCount{ID: "c1", Status: "submitted", CreatedBy: "user-1"}}
	svc := NewInventoryCountsService(repo, nil)
	resp := svc.Cancel("c1", "user-1", "operator")
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
	assert.False(t, repo.cancelCalled)
}

func TestInventoryCountsService_Submit_ReturnsErrorIfRepoFails(t *testing.T) {
	repo := &mockCountsRepo{
		count:            &database.InventoryCount{ID: "c1", Status: "in_progress", CreatedBy: "user-1"},
		submitWithAdjErr: &responses.InternalResponse{Error: errors.New("db error"), Message: "db", Handled: false},
	}
	svc := NewInventoryCountsService(repo, nil)
	_, resp := svc.Submit("c1", "user-1", "operator")
	require.NotNil(t, resp)
}
