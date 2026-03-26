package services

import (
	"errors"
	"fmt"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// mockArticlesRepo is a in-memory fake for unit testing ArticlesService.
type mockArticlesRepo struct {
	articles   []database.Article
	byID       map[string]*database.Article
	bySku      map[string]*database.Article
	createErr  *responses.InternalResponse
	getIDErr   *responses.InternalResponse
	deleteErr  *responses.InternalResponse
	lotsBySku  []database.Lot
	serialsBySku []database.Serial
}

func (m *mockArticlesRepo) GetAllArticles() ([]database.Article, *responses.InternalResponse) {
	if m.articles == nil {
		return nil, nil
	}
	return m.articles, nil
}

func (m *mockArticlesRepo) GetArticleByID(id string) (*database.Article, *responses.InternalResponse) {
	if m.getIDErr != nil {
		return nil, m.getIDErr
	}
	if m.byID != nil {
		if a, ok := m.byID[id]; ok {
			return a, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Artículo no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockArticlesRepo) GetBySku(sku string) (*database.Article, *responses.InternalResponse) {
	if m.bySku != nil {
		if a, ok := m.bySku[sku]; ok {
			return a, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "Artículo no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockArticlesRepo) CreateArticle(data *requests.Article) *responses.InternalResponse {
	if m.createErr != nil {
		return m.createErr
	}
	id := fmt.Sprintf("art-%d", len(m.articles)+1)
	a := database.Article{
		ID:              id,
		SKU:             data.SKU,
		Name:            data.Name,
		Description:     data.Description,
		UnitPrice:       data.UnitPrice,
		Presentation:    data.Presentation,
		TrackByLot:      data.TrackByLot,
		TrackBySerial:   data.TrackBySerial,
		TrackExpiration: data.TrackExpiration,
		MinQuantity:     data.MinQuantity,
		MaxQuantity:     data.MaxQuantity,
		ImageURL:        data.ImageURL,
	}
	m.articles = append(m.articles, a)
	return nil
}

func (m *mockArticlesRepo) UpdateArticle(id string, data *requests.Article) (*database.Article, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockArticlesRepo) GetLotsBySKU(sku string) ([]database.Lot, error) {
	if m.lotsBySku != nil {
		return m.lotsBySku, nil
	}
	return nil, nil
}

func (m *mockArticlesRepo) GetSerialsBySKU(sku string) ([]database.Serial, error) {
	if m.serialsBySku != nil {
		return m.serialsBySku, nil
	}
	return nil, nil
}

func (m *mockArticlesRepo) ImportArticlesFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return nil, nil
}

func (m *mockArticlesRepo) ExportArticlesToExcel() ([]byte, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockArticlesRepo) GenerateImportTemplate(_ string) ([]byte, *responses.InternalResponse) {
	return nil, nil
}

func (m *mockArticlesRepo) DeleteArticle(id string) *responses.InternalResponse {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func TestArticlesService_GetAllArticles(t *testing.T) {
	repo := &mockArticlesRepo{
		articles: []database.Article{
			{ID: "1", SKU: "SKU1", Name: "Art1", Presentation: "unit"},
			{ID: "2", SKU: "SKU2", Name: "Art2", Presentation: "unit"},
		},
	}
	svc := NewArticlesService(repo)
	list, errResp := svc.GetAllArticles()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "SKU1", list[0].SKU)
	assert.Equal(t, "SKU2", list[1].SKU)
}

func TestArticlesService_GetArticleByID_NotFound(t *testing.T) {
	repo := &mockArticlesRepo{byID: map[string]*database.Article{}}
	svc := NewArticlesService(repo)
	art, errResp := svc.GetArticleByID("99")
	require.NotNil(t, errResp)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
	assert.Nil(t, art)
}

func TestArticlesService_GetArticleByID_Found(t *testing.T) {
	repo := &mockArticlesRepo{
		byID: map[string]*database.Article{
			"1": {ID: "1", SKU: "SKU1", Name: "Art1", Presentation: "unit"},
		},
	}
	svc := NewArticlesService(repo)
	art, errResp := svc.GetArticleByID("1")
	require.Nil(t, errResp)
	require.NotNil(t, art)
	assert.Equal(t, "SKU1", art.SKU)
}

func TestArticlesService_CreateArticle_Success(t *testing.T) {
	repo := &mockArticlesRepo{articles: []database.Article{}}
	svc := NewArticlesService(repo)
	req := &requests.Article{
		SKU:          "NEW-SKU",
		Name:         "New Article",
		Presentation: "unit",
	}
	errResp := svc.CreateArticle(req)
	require.Nil(t, errResp)
	require.Len(t, repo.articles, 1)
	assert.Equal(t, "NEW-SKU", repo.articles[0].SKU)
}

func TestArticlesService_CreateArticle_Conflict(t *testing.T) {
	repo := &mockArticlesRepo{
		createErr: &responses.InternalResponse{
			Message:    "Ya existe un artículo con el mismo SKU",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewArticlesService(repo)
	req := &requests.Article{SKU: "DUP", Name: "Dup", Presentation: "unit"}
	errResp := svc.CreateArticle(req)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestArticlesService_DeleteArticle_Success(t *testing.T) {
	repo := &mockArticlesRepo{}
	svc := NewArticlesService(repo)
	errResp := svc.DeleteArticle("1")
	require.Nil(t, errResp)
}

func TestArticlesService_DeleteArticle_Error(t *testing.T) {
	repo := &mockArticlesRepo{
		deleteErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error al eliminar",
			Handled: false,
		},
	}
	svc := NewArticlesService(repo)
	errResp := svc.DeleteArticle("1")
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}
