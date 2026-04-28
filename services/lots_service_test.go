package services

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// mockLotsRepo is an in-memory fake for unit testing LotsService.
type mockLotsRepo struct {
	lots      []database.Lot
	getAllErr *responses.InternalResponse
	createErr *responses.InternalResponse
}

func (m *mockLotsRepo) GetAllLots() ([]database.Lot, *responses.InternalResponse) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	if m.lots == nil {
		return nil, nil
	}
	return m.lots, nil
}

func (m *mockLotsRepo) GetLotsBySKU(sku *string) ([]database.Lot, *responses.InternalResponse) {
	if m.lots == nil {
		return nil, nil
	}
	if sku == nil || *sku == "" {
		return m.lots, nil
	}
	var out []database.Lot
	for _, l := range m.lots {
		if l.SKU == *sku {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *mockLotsRepo) CreateLot(data *requests.CreateLotRequest) *responses.InternalResponse {
	if m.createErr != nil {
		return m.createErr
	}
	return nil
}

func (m *mockLotsRepo) UpdateLot(id string, data map[string]interface{}) *responses.InternalResponse {
	return nil
}

func (m *mockLotsRepo) DeleteLot(id string) *responses.InternalResponse {
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

func (m *mockLotsRepo) GetLotTrace(_ string) (*responses.LotTraceResponse, *responses.InternalResponse) {
	return nil, nil
}

// mockArticlesRepoForLots returns a fixed article for GetBySku for rotation-order tests.
type mockArticlesRepoForLots struct {
	rotationStrategy string
}

func (m *mockArticlesRepoForLots) GetBySku(sku string) (*database.Article, *responses.InternalResponse) {
	if m == nil {
		return nil, &responses.InternalResponse{Message: "no repo"}
	}
	return &database.Article{SKU: sku, RotationStrategy: m.rotationStrategy}, nil
}

func (m *mockArticlesRepoForLots) GetAllArticles() ([]database.Article, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GetArticleByID(id string) (*database.Article, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) CreateArticle(data *requests.Article) *responses.InternalResponse {
	return nil
}
func (m *mockArticlesRepoForLots) UpdateArticle(id string, data *requests.Article) (*database.Article, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GetLotsBySKU(sku string) ([]database.Lot, error) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) GetSerialsBySKU(sku string) ([]database.Serial, error) {
	return nil, nil
}
func (m *mockArticlesRepoForLots) ImportArticlesFromExcel(_ []byte) ([]string, []string, []*responses.InternalResponse) {
	return nil, nil, nil
}

func (m *mockArticlesRepoForLots) ImportArticlesFromJSON(_ []requests.ArticleImportRow) ([]string, []string, []*responses.InternalResponse) {
	return nil, nil, nil
}
func (m *mockArticlesRepoForLots) ExportArticlesToExcel() ([]byte, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockArticlesRepoForLots) GenerateImportTemplate(_ string) ([]byte, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockArticlesRepoForLots) ValidateImportRows(_ []requests.ArticleImportRow) ([]responses.ArticleValidationResult, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockArticlesRepoForLots) DeleteArticle(id string) *responses.InternalResponse {
	return nil
}

func TestLotsService_GetAllLots(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", LotNumber: "L1", SKU: "SKU-A", Quantity: 10},
			{ID: "2", LotNumber: "L2", SKU: "SKU-B", Quantity: 20},
		},
	}
	svc := NewLotsService(repo, nil)
	list, errResp := svc.GetAllLots()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "SKU-A", list[0].SKU)
	assert.Equal(t, "SKU-B", list[1].SKU)
}

func TestLotsService_GetLotsBySKU_NilSku_ReturnsAll(t *testing.T) {
	repo := &mockLotsRepo{
		lots: []database.Lot{
			{ID: "1", SKU: "S1", Quantity: 1},
			{ID: "2", SKU: "S2", Quantity: 2},
		},
	}
	svc := NewLotsService(repo, nil)
	list, errResp := svc.GetLotsBySKU(nil)
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
	list, errResp := svc.GetLotsBySKU(&sku)
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "S1", list[0].SKU)
	assert.Equal(t, "S1", list[1].SKU)
}

func TestLotsService_CreateLot_Success(t *testing.T) {
	repo := &mockLotsRepo{}
	svc := NewLotsService(repo, nil)
	req := &requests.CreateLotRequest{
		LotNumber: "LOT-001",
		SKU:       "ART-1",
		Quantity:  100,
	}
	errResp := svc.Create(req)
	require.Nil(t, errResp)
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
	errResp := svc.Create(req)
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
	list, errResp := svc.GetLotsBySKU(&sku)
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
	list, errResp := svc.GetLotsBySKU(&sku)
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
	errResp := svc.Create(req)
	require.Nil(t, errResp)
}

func TestLotsService_GetLotByID_Found(t *testing.T) {
	now := time.Now()
	repo := &mockLotsRepo{lots: []database.Lot{
		{ID: "lot-abc", LotNumber: "L1", SKU: "SKU-001", CreatedAt: now},
	}}
	svc := NewLotsService(repo, nil)
	lot, errResp := svc.GetLotByID("lot-abc")
	require.Nil(t, errResp)
	require.NotNil(t, lot)
	assert.Equal(t, "L1", lot.LotNumber)
}

func TestLotsService_GetLotByID_NotFound(t *testing.T) {
	repo := &mockLotsRepo{}
	svc := NewLotsService(repo, nil)
	_, errResp := svc.GetLotByID("nonexistent")
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
	list, _ := svc.GetLotsBySKU(&sku)
	require.Len(t, list, 2)
	// FEFO by expiration_date: L2 (5 days) before L1 (10 days)
	assert.Equal(t, "L2", list[0].LotNumber)
	assert.Equal(t, "L1", list[1].LotNumber)
}
