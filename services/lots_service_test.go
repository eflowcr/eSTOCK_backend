package services

import (
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tenant constants used by the tests. The "other" tenant is used to assert isolation.
const (
	testTenantA = "00000000-0000-0000-0000-00000000000a"
	testTenantB = "00000000-0000-0000-0000-00000000000b"
)

// mockLotsRepo is an in-memory fake for unit testing LotsService.
//
// S3.5 W2-B: filters every read by tenantID so tests can assert isolation between
// tenants (Get/Create/Update/Delete from tenant A must not affect tenant B).
type mockLotsRepo struct {
	lots             []database.Lot
	getAllErr        *responses.InternalResponse
	createErr        *responses.InternalResponse
	lastCreateTenant string
	lastUpdateTenant string
	lastDeleteTenant string
}

func (m *mockLotsRepo) GetAllLots(tenantID string) ([]database.Lot, *responses.InternalResponse) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	if m.lots == nil {
		return nil, nil
	}
	out := make([]database.Lot, 0, len(m.lots))
	for _, l := range m.lots {
		if l.TenantID == "" || l.TenantID == tenantID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *mockLotsRepo) GetLotsBySKU(tenantID string, sku *string) ([]database.Lot, *responses.InternalResponse) {
	if m.lots == nil {
		return nil, nil
	}
	out := make([]database.Lot, 0, len(m.lots))
	for _, l := range m.lots {
		if l.TenantID != "" && l.TenantID != tenantID {
			continue
		}
		if sku == nil || *sku == "" || l.SKU == *sku {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *mockLotsRepo) CreateLot(tenantID string, data *requests.CreateLotRequest) *responses.InternalResponse {
	m.lastCreateTenant = tenantID
	if m.createErr != nil {
		return m.createErr
	}
	return nil
}

func (m *mockLotsRepo) UpdateLot(tenantID, id string, data map[string]interface{}) *responses.InternalResponse {
	m.lastUpdateTenant = tenantID
	return nil
}

func (m *mockLotsRepo) DeleteLot(tenantID, id string) *responses.InternalResponse {
	m.lastDeleteTenant = tenantID
	return nil
}

func (m *mockLotsRepo) GetLotByID(id string) (*database.Lot, *responses.InternalResponse) {
	for i := range m.lots {
		if m.lots[i].ID == id {
			return &m.lots[i], nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockLotsRepo) GetLotByIDForTenant(id, tenantID string) (*database.Lot, *responses.InternalResponse) {
	for i := range m.lots {
		if m.lots[i].ID == id && (m.lots[i].TenantID == "" || m.lots[i].TenantID == tenantID) {
			return &m.lots[i], nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockLotsRepo) GetLotTrace(_ string, _ string) (*responses.LotTraceResponse, *responses.InternalResponse) {
	return nil, nil
}

// mockArticlesRepoForLots returns a fixed article for GetBySku for rotation-order tests.
type mockArticlesRepoForLots struct {
	rotationStrategy string
}

// S3.5 W1 — interface bumped to include ForTenant variants. Lots service only
// reads via GetBySku (legacy non-tenant path; lots/serials still use SKU-only FK
// resolution until W2 retrofits child tables).

func (m *mockArticlesRepoForLots) GetBySku(sku string) (*database.Article, *responses.InternalResponse) {
	if m == nil {
		return nil, &responses.InternalResponse{Message: "no repo"}
	}
	return &database.Article{SKU: sku, RotationStrategy: m.rotationStrategy}, nil
}

// ── tenant-scoped (no-op) ───────────────────────────────────────────────────

func (m *mockArticlesRepoForLots) GetAllArticlesForTenant(_ string) ([]database.Article, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GetArticleByIDForTenant(_, _ string) (*database.Article, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GetBySkuForTenant(sku, _ string) (*database.Article, *responses.InternalResponse) {
	return m.GetBySku(sku)
}
func (m *mockArticlesRepoForLots) CreateArticleForTenant(_ string, _ *requests.Article) *responses.InternalResponse {
	return nil
}
func (m *mockArticlesRepoForLots) UpdateArticleForTenant(_, _ string, _ *requests.Article) (*database.Article, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) DeleteArticleForTenant(_, _ string) *responses.InternalResponse {
	return nil
}
func (m *mockArticlesRepoForLots) ImportArticlesFromExcelForTenant(_ string, _ []byte) ([]string, []string, []*responses.InternalResponse) {
	return nil, nil, nil
}
func (m *mockArticlesRepoForLots) ImportArticlesFromJSONForTenant(_ string, _ []requests.ArticleImportRow) ([]string, []string, []*responses.InternalResponse) {
	return nil, nil, nil
}
func (m *mockArticlesRepoForLots) ValidateImportRowsForTenant(_ string, _ []requests.ArticleImportRow) ([]responses.ArticleValidationResult, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) ExportArticlesToExcelForTenant(_ string) ([]byte, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GenerateImportTemplateForTenant(_, _ string) ([]byte, *responses.InternalResponse) {
	return nil, nil
}

// ── legacy non-tenant ───────────────────────────────────────────────────────

func (m *mockArticlesRepoForLots) GetAllArticles() ([]database.Article, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GetArticleByID(_ string) (*database.Article, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GetLotsBySKU(_ string) ([]database.Lot, error) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GetSerialsBySKU(_ string) ([]database.Serial, error) {
	return nil, nil
}

func TestLotsService_GetAllLots(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", LotNumber: "L1", SKU: "SKU-A", Quantity: 10},
			{ID: "2", LotNumber: "L2", SKU: "SKU-B", Quantity: 20},
		},
	}
	svc := NewLotsService(repo, nil)
	list, errResp := svc.GetAllLots(testTenantA)
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "SKU-A", list[0].SKU)
	assert.Equal(t, "SKU-B", list[1].SKU)
}

// S3.5 W2-B: GetAllLots must not return rows owned by another tenant.
func TestLotsService_GetAllLots_TenantIsolation(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", LotNumber: "L1", SKU: "SKU-A", Quantity: 10, TenantID: testTenantA},
			{ID: "2", LotNumber: "L2", SKU: "SKU-B", Quantity: 20, TenantID: testTenantB},
		},
	}
	svc := NewLotsService(repo, nil)
	listA, errResp := svc.GetAllLots(testTenantA)
	require.Nil(t, errResp)
	require.Len(t, listA, 1)
	assert.Equal(t, "SKU-A", listA[0].SKU)

	listB, errResp := svc.GetAllLots(testTenantB)
	require.Nil(t, errResp)
	require.Len(t, listB, 1)
	assert.Equal(t, "SKU-B", listB[0].SKU)
}

func TestLotsService_GetLotsBySKU_NilSku_ReturnsAll(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", SKU: "S1", Quantity: 1},
			{ID: "2", SKU: "S2", Quantity: 2},
		},
	}
	svc := NewLotsService(repo, nil)
	list, errResp := svc.GetLotsBySKU(testTenantA, nil)
	require.Nil(t, errResp)
	require.Len(t, list, 2)
}

func TestLotsService_GetLotsBySKU_Filtered(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", SKU: "S1", Quantity: 1},
			{ID: "2", SKU: "S1", Quantity: 2},
			{ID: "3", SKU: "S2", Quantity: 3},
		},
	}
	svc := NewLotsService(repo, nil)
	sku := "S1"
	list, errResp := svc.GetLotsBySKU(testTenantA, &sku)
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "S1", list[0].SKU)
	assert.Equal(t, "S1", list[1].SKU)
}

// S3.5 W2-B: same SKU exists for two tenants — service must only return caller's tenant's lots.
func TestLotsService_GetLotsBySKU_TenantIsolation(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", SKU: "SHARED", LotNumber: "A-L1", TenantID: testTenantA},
			{ID: "2", SKU: "SHARED", LotNumber: "B-L1", TenantID: testTenantB},
		},
	}
	svc := NewLotsService(repo, nil)
	sku := "SHARED"
	listA, errResp := svc.GetLotsBySKU(testTenantA, &sku)
	require.Nil(t, errResp)
	require.Len(t, listA, 1)
	assert.Equal(t, "A-L1", listA[0].LotNumber)
}

func TestLotsService_CreateLot_Success(t *testing.T) {
	repo := &mockLotsRepo{}
	svc := NewLotsService(repo, nil)
	req := &requests.CreateLotRequest{
		LotNumber: "LOT-001",
		SKU:       "ART-1",
		Quantity:  100,
	}
	errResp := svc.Create(testTenantA, req)
	require.Nil(t, errResp)
	assert.Equal(t, testTenantA, repo.lastCreateTenant, "tenantID must be threaded through to repo")
}

func TestLotsService_CreateLot_Error(t *testing.T) {
	repo := &mockLotsRepo{
		createErr: &responses.InternalResponse{
			Message: "Failed to create lot",
			Handled: false,
		},
	}
	svc := NewLotsService(repo, nil)
	req := &requests.CreateLotRequest{LotNumber: "L", SKU: "S", Quantity: 1}
	errResp := svc.Create(testTenantA, req)
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}

func TestLotsService_GetLotsBySKU_OrdersByFIFO(t *testing.T) {
	t1 := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", SKU: "S1", LotNumber: "L1", CreatedAt: t1},
			{ID: "2", SKU: "S1", LotNumber: "L2", CreatedAt: t2},
			{ID: "3", SKU: "S1", LotNumber: "L3", CreatedAt: t3},
		},
	}
	articlesRepo := &mockArticlesRepoForLots{rotationStrategy: "fifo"}
	svc := NewLotsService(repo, articlesRepo)
	sku := "S1"
	list, errResp := svc.GetLotsBySKU(testTenantA, &sku)
	require.Nil(t, errResp)
	require.Len(t, list, 3)
	// FIFO: oldest first -> L3 (t3), L1 (t1), L2 (t2)
	assert.Equal(t, "L3", list[0].LotNumber)
	assert.Equal(t, "L1", list[1].LotNumber)
	assert.Equal(t, "L2", list[2].LotNumber)
}

func TestLotsService_GetLotsBySKU_OrdersByFEFO(t *testing.T) {
	exp1 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	exp2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", SKU: "S1", LotNumber: "L1", ExpirationDate: &exp1, CreatedAt: time.Now()},
			{ID: "2", SKU: "S1", LotNumber: "L2", ExpirationDate: &exp2, CreatedAt: time.Now()},
		},
	}
	articlesRepo := &mockArticlesRepoForLots{rotationStrategy: "fefo"}
	svc := NewLotsService(repo, articlesRepo)
	sku := "S1"
	list, errResp := svc.GetLotsBySKU(testTenantA, &sku)
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	// FEFO: earliest expiry first -> L2 (exp2), L1 (exp1)
	assert.Equal(t, "L2", list[0].LotNumber)
	assert.Equal(t, "L1", list[1].LotNumber)
}

// ─── L1: Extended lot fields ──────────────────────────────────────────────────

func TestLotsService_CreateLot_WithExtendedFields(t *testing.T) {
	repo := &mockLotsRepo{}
	svc := NewLotsService(repo, nil)
	notes := "test lot notes"
	mfgDate := "2026-01-15"
	bbd := "2026-12-31"
	req := &requests.CreateLotRequest{
		LotNumber:      "LOT-EXT-001",
		SKU:            "SKU-001",
		Quantity:       100,
		LotNotes:       &notes,
		ManufacturedAt: &mfgDate,
		BestBeforeDate: &bbd,
	}
	errResp := svc.Create(testTenantA, req)
	require.Nil(t, errResp)
}

func TestLotsService_GetLotByID_Found(t *testing.T) {
	now := time.Now()
	repo := &mockLotsRepo{lots: []database.Lot{
		{ID: "lot-abc", LotNumber: "L1", SKU: "SKU-001", CreatedAt: now},
	}}
	svc := NewLotsService(repo, nil)
	lot, errResp := svc.GetLotByID(testTenantA, "lot-abc")
	require.Nil(t, errResp)
	require.NotNil(t, lot)
	assert.Equal(t, "L1", lot.LotNumber)
}

func TestLotsService_GetLotByID_NotFound(t *testing.T) {
	repo := &mockLotsRepo{}
	svc := NewLotsService(repo, nil)
	_, errResp := svc.GetLotByID(testTenantA, "nonexistent")
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

// S3.5 W2-B: lot belongs to tenant B; tenant A must see NotFound (no info leak).
func TestLotsService_GetLotByID_CrossTenantReturnsNotFound(t *testing.T) {
	now := time.Now()
	repo := &mockLotsRepo{lots: []database.Lot{
		{ID: "lot-b", LotNumber: "B-L1", SKU: "SKU-001", CreatedAt: now, TenantID: testTenantB},
	}}
	svc := NewLotsService(repo, nil)
	_, errResp := svc.GetLotByID(testTenantA, "lot-b")
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

// FEFO regression: lots MUST order by expiration_date, NOT best_before_date
func TestLotsService_FEFO_UsesExpirationDate_NotBBD(t *testing.T) {
	now := time.Now()
	exp1 := now.Add(10 * 24 * time.Hour) // expires in 10 days
	exp2 := now.Add(5 * 24 * time.Hour)  // expires in 5 days (should come first in FEFO)
	bbd1 := now.Add(20 * 24 * time.Hour) // BBD doesn't matter for FEFO ordering
	repo := &mockLotsRepo{lots: []database.Lot{
		{ID: "l1", LotNumber: "L1", SKU: "SKU-FEFO", ExpirationDate: &exp1, BestBeforeDate: &bbd1, CreatedAt: now},
		{ID: "l2", LotNumber: "L2", SKU: "SKU-FEFO", ExpirationDate: &exp2, CreatedAt: now},
	}}
	sku := "SKU-FEFO"
	artRepo := &mockArticlesRepoForLots{rotationStrategy: "fefo"}
	svc := NewLotsService(repo, artRepo)
	list, _ := svc.GetLotsBySKU(testTenantA, &sku)
	require.Len(t, list, 2)
	// FEFO by expiration_date: L2 (5 days) before L1 (10 days)
	assert.Equal(t, "L2", list[0].LotNumber)
	assert.Equal(t, "L1", list[1].LotNumber)
}

// S3.5 W2-B: Update/Delete propagate tenantID to the repo.
func TestLotsService_UpdateAndDelete_PropagateTenantID(t *testing.T) {
	repo := &mockLotsRepo{}
	svc := NewLotsService(repo, nil)
	require.Nil(t, svc.UpdateUpdateLot(testTenantA, "lot-1", map[string]interface{}{"quantity": 5.0}))
	assert.Equal(t, testTenantA, repo.lastUpdateTenant)
	require.Nil(t, svc.DeleteLot(testTenantA, "lot-1"))
	assert.Equal(t, testTenantA, repo.lastDeleteTenant)
}
