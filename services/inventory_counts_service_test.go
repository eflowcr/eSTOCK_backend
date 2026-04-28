package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/dto"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── mock InventoryCountsRepository ──────────────────────────────────────────

type mockCountsRepo struct {
	count             *database.InventoryCount
	getErr            *responses.InternalResponse
	listLines         []database.InventoryCountLine
	listLinesErr      *responses.InternalResponse
	addLineErr        *responses.InternalResponse
	addedLines        []database.InventoryCountLine
	resolvedSKU       string
	resolveErr        *responses.InternalResponse
	expectedQty       float64
	expectedQtyErr    *responses.InternalResponse
	locationCode      string
	locationErr       *responses.InternalResponse
	startCalled       bool
	cancelCalled      bool
	submitCalled      bool
	submittedAdjID    string
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

// ─── mock AdjustmentsRepository ──────────────────────────────────────────────

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

func TestInventoryCountsService_Submit_CreatesAdjustments(t *testing.T) {
	repo := &mockCountsRepo{
		count:        &database.InventoryCount{ID: "c1", Code: "CC-001", Status: "in_progress"},
		locationCode: "LOC-A",
		listLines: []database.InventoryCountLine{
			{ID: "l1", CountID: "c1", LocationID: "loc-1", SKU: "SKU-1", ExpectedQty: 10, ScannedQty: 12, VarianceQty: 2},
			{ID: "l2", CountID: "c1", LocationID: "loc-1", SKU: "SKU-2", ExpectedQty: 5, ScannedQty: 5, VarianceQty: 0},
			{ID: "l3", CountID: "c1", LocationID: "loc-1", SKU: "SKU-3", ExpectedQty: 8, ScannedQty: 4, VarianceQty: -4},
		},
	}
	adjRepo := &mockAdjustmentsRepoForCounts{}
	svc := NewInventoryCountsService(repo, adjRepo)
	updated, resp := svc.Submit("c1", "user-1")
	require.Nil(t, resp)
	require.NotNil(t, updated)
	assert.Equal(t, "submitted", updated.Status)
	// Two non-zero variance lines → two adjustments
	require.Len(t, adjRepo.createCalls, 2)
	assert.Equal(t, "INVENTORY_COUNT_INBOUND", adjRepo.createCalls[0].Reason)
	assert.Equal(t, 2.0, adjRepo.createCalls[0].AdjustmentQuantity)
	assert.Equal(t, "INVENTORY_COUNT_OUTBOUND", adjRepo.createCalls[1].Reason)
	assert.Equal(t, 4.0, adjRepo.createCalls[1].AdjustmentQuantity)
	assert.True(t, repo.submitCalled)
	assert.Equal(t, "adj-1", repo.submittedAdjID)
}

func TestInventoryCountsService_Submit_NoVarianceNoAdjustment(t *testing.T) {
	repo := &mockCountsRepo{
		count: &database.InventoryCount{ID: "c1", Code: "CC-001", Status: "in_progress"},
		listLines: []database.InventoryCountLine{
			{ID: "l1", CountID: "c1", SKU: "SKU-1", ExpectedQty: 10, ScannedQty: 10, VarianceQty: 0},
		},
		locationCode: "LOC-A",
	}
	adjRepo := &mockAdjustmentsRepoForCounts{}
	svc := NewInventoryCountsService(repo, adjRepo)
	_, resp := svc.Submit("c1", "user-1")
	require.Nil(t, resp)
	assert.Empty(t, adjRepo.createCalls)
	assert.True(t, repo.submitCalled)
}

func TestInventoryCountsService_Cancel(t *testing.T) {
	repo := &mockCountsRepo{count: &database.InventoryCount{ID: "c1", Status: "in_progress"}}
	svc := NewInventoryCountsService(repo, nil)
	resp := svc.Cancel("c1")
	require.Nil(t, resp)
	assert.True(t, repo.cancelCalled)
	assert.Equal(t, "cancelled", repo.count.Status)
}

func TestInventoryCountsService_Cancel_RejectsClosed(t *testing.T) {
	repo := &mockCountsRepo{count: &database.InventoryCount{ID: "c1", Status: "submitted"}}
	svc := NewInventoryCountsService(repo, nil)
	resp := svc.Cancel("c1")
	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, resp.StatusCode)
	assert.False(t, repo.cancelCalled)
}

func TestInventoryCountsService_Submit_ReturnsErrorIfRepoFails(t *testing.T) {
	repo := &mockCountsRepo{
		count:        &database.InventoryCount{ID: "c1", Status: "in_progress"},
		listLinesErr: &responses.InternalResponse{Error: errors.New("db error"), Message: "db", Handled: false},
	}
	svc := NewInventoryCountsService(repo, nil)
	_, resp := svc.Submit("c1", "user-1")
	require.NotNil(t, resp)
}
