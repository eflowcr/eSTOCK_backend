package services

import (
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── ImportArticlesFromJSON ────────────────────────────────────────────────────

func TestArticlesService_ImportJSON_Success(t *testing.T) {
	repo := &mockArticlesRepo{}
	svc := NewArticlesService(repo)

	rows := []requests.ArticleImportRow{
		{SKU: "SKU-001", Name: "Product One", Presentation: "unit"},
		{SKU: "SKU-002", Name: "Product Two", Presentation: "box", TrackByLot: "Si"},
	}

	imported, skipped, errs := svc.ImportArticlesFromJSON(rows)
	assert.Empty(t, errs)
	assert.Empty(t, skipped)
	assert.Len(t, imported, 0) // mock returns nil; real repo would return skus
}

func TestArticlesService_ImportJSON_EmptyRows(t *testing.T) {
	repo := &mockArticlesRepo{}
	svc := NewArticlesService(repo)

	imported, skipped, errs := svc.ImportArticlesFromJSON([]requests.ArticleImportRow{})
	assert.Empty(t, errs)
	assert.Empty(t, skipped)
	assert.Empty(t, imported)
}

// ── ValidateImportRows ────────────────────────────────────────────────────────

func TestArticlesService_ValidateImportRows_Delegates(t *testing.T) {
	repo := &mockArticlesRepo{}
	svc := NewArticlesService(repo)

	rows := []requests.ArticleImportRow{
		{SKU: "A", Name: "Alfa", Presentation: "unit"},
	}
	results, errResp := svc.ValidateImportRows(rows)
	// mock returns nil,nil — just verifies delegation doesn't panic
	assert.Nil(t, errResp)
	assert.Nil(t, results)
}

// ── validateRotationStrategy (internal service rule) ─────────────────────────

func TestArticlesService_RotationStrategy_FEFORequiresExpiration(t *testing.T) {
	repo := &mockArticlesRepo{}
	svc := NewArticlesService(repo)

	req := &requests.Article{
		SKU:              "TEST",
		Name:             "Test",
		Presentation:     "unit",
		TrackExpiration:  false,
		RotationStrategy: "fefo",
	}
	errResp := svc.CreateArticle(req)
	require.NotNil(t, errResp)
	assert.True(t, errResp.Handled)
	assert.Contains(t, errResp.Message, "FEFO")
}

func TestArticlesService_RotationStrategy_FIFONoExpiration(t *testing.T) {
	repo := &mockArticlesRepo{}
	svc := NewArticlesService(repo)

	req := &requests.Article{
		SKU:              "TEST",
		Name:             "Test",
		Presentation:     "unit",
		TrackExpiration:  false,
		RotationStrategy: "fifo",
	}
	errResp := svc.CreateArticle(req)
	assert.Nil(t, errResp)
}

func TestArticlesService_RotationStrategy_FEFOWithExpiration(t *testing.T) {
	repo := &mockArticlesRepo{}
	svc := NewArticlesService(repo)

	req := &requests.Article{
		SKU:              "TEST",
		Name:             "Test",
		Presentation:     "unit",
		TrackExpiration:  true,
		RotationStrategy: "fefo",
	}
	errResp := svc.CreateArticle(req)
	assert.Nil(t, errResp)
}

// ── UpdateArticle — lot/serial tracking warnings ──────────────────────────────

func TestArticlesService_UpdateArticle_LotTrackingDisabledWarning(t *testing.T) {
	existing := database.Article{ID: "1", SKU: "SKU1", Name: "Art", Presentation: "unit", TrackByLot: true}
	repo := &mockArticlesRepo{
		byID:      map[string]*database.Article{"1": &existing},
		lotsBySku: []database.Lot{{ID: "lot1", LotNumber: "L001", SKU: "SKU1"}},
	}
	svc := NewArticlesService(repo)

	req := &requests.Article{SKU: "SKU1", Name: "Art", Presentation: "unit", TrackByLot: false}
	_, errResp, warnings := svc.UpdateArticle("1", req)
	assert.Nil(t, errResp)
	require.Len(t, warnings, 1)
	assert.Equal(t, "lot_tracking_disabled", warnings[0]["type"])
}

func TestArticlesService_UpdateArticle_SerialTrackingDisabledWarning(t *testing.T) {
	existing := database.Article{ID: "1", SKU: "SKU1", Name: "Art", Presentation: "unit", TrackBySerial: true}
	repo := &mockArticlesRepo{
		byID:         map[string]*database.Article{"1": &existing},
		serialsBySku: []database.Serial{{ID: "s1", SerialNumber: "SN-001", SKU: "SKU1"}},
	}
	svc := NewArticlesService(repo)

	req := &requests.Article{SKU: "SKU1", Name: "Art", Presentation: "unit", TrackBySerial: false}
	_, errResp, warnings := svc.UpdateArticle("1", req)
	assert.Nil(t, errResp)
	require.Len(t, warnings, 1)
	assert.Equal(t, "serial_tracking_disabled", warnings[0]["type"])
}

func TestArticlesService_UpdateArticle_NotFound(t *testing.T) {
	repo := &mockArticlesRepo{byID: map[string]*database.Article{}}
	svc := NewArticlesService(repo)

	_, errResp, _ := svc.UpdateArticle("999", &requests.Article{SKU: "X", Name: "X", Presentation: "unit"})
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}
