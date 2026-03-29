package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ─── mock repo ───────────────────────────────────────────────────────────────

type mockArticlesRepoCtrl struct {
	articles  []database.Article
	byID      map[string]*database.Article
	bySku     map[string]*database.Article
	createErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
}

func (m *mockArticlesRepoCtrl) GetAllArticles() ([]database.Article, *responses.InternalResponse) {
	return m.articles, nil
}
func (m *mockArticlesRepoCtrl) GetArticleByID(id string) (*database.Article, *responses.InternalResponse) {
	if m.byID != nil {
		if a, ok := m.byID[id]; ok {
			return a, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}
func (m *mockArticlesRepoCtrl) GetBySku(sku string) (*database.Article, *responses.InternalResponse) {
	if m.bySku != nil {
		if a, ok := m.bySku[sku]; ok {
			return a, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}
func (m *mockArticlesRepoCtrl) CreateArticle(data *requests.Article) *responses.InternalResponse {
	return m.createErr
}
func (m *mockArticlesRepoCtrl) UpdateArticle(id string, data *requests.Article) (*database.Article, *responses.InternalResponse) {
	if m.byID != nil {
		if a, ok := m.byID[id]; ok {
			return a, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}
func (m *mockArticlesRepoCtrl) GetLotsBySKU(sku string) ([]database.Lot, error)     { return nil, nil }
func (m *mockArticlesRepoCtrl) GetSerialsBySKU(sku string) ([]database.Serial, error) { return nil, nil }
func (m *mockArticlesRepoCtrl) ImportArticlesFromExcel(fileBytes []byte) ([]string, []string, []*responses.InternalResponse) {
	return nil, nil, nil
}
func (m *mockArticlesRepoCtrl) ImportArticlesFromJSON(_ []requests.ArticleImportRow) ([]string, []string, []*responses.InternalResponse) {
	return nil, nil, nil
}
func (m *mockArticlesRepoCtrl) ExportArticlesToExcel() ([]byte, *responses.InternalResponse) {
	return []byte("xlsx"), nil
}
func (m *mockArticlesRepoCtrl) GenerateImportTemplate(_ string) ([]byte, *responses.InternalResponse) {
	return []byte("tpl"), nil
}
func (m *mockArticlesRepoCtrl) ValidateImportRows(_ []requests.ArticleImportRow) ([]responses.ArticleValidationResult, *responses.InternalResponse) {
	return nil, nil
}
func (m *mockArticlesRepoCtrl) DeleteArticle(id string) *responses.InternalResponse {
	return m.deleteErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newArticlesController(repo *mockArticlesRepoCtrl) *ArticlesController {
	svc := services.NewArticlesService(repo)
	return NewArticlesController(*svc, nil, nil)
}

func performRequest(handler gin.HandlerFunc, method, path string, body interface{}, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	if params != nil {
		c.Params = params
	}
	handler(c)
	return w
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestArticlesController_GetAllArticles_Empty(t *testing.T) {
	ctrl := newArticlesController(&mockArticlesRepoCtrl{articles: []database.Article{}})
	w := performRequest(ctrl.GetAllArticles, "GET", "/articles", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticlesController_GetAllArticles_WithData(t *testing.T) {
	repo := &mockArticlesRepoCtrl{
		articles: []database.Article{{ID: "1", SKU: "SKU1", Name: "Art1", Presentation: "unit"}},
	}
	ctrl := newArticlesController(repo)
	w := performRequest(ctrl.GetAllArticles, "GET", "/articles", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Result.Success)
}

func TestArticlesController_GetArticleByID_Found(t *testing.T) {
	repo := &mockArticlesRepoCtrl{
		byID: map[string]*database.Article{"art-1": {ID: "art-1", SKU: "SKU1", Name: "Art1", Presentation: "unit"}},
	}
	ctrl := newArticlesController(repo)
	w := performRequest(ctrl.GetArticleByID, "GET", "/articles/art-1", nil, gin.Params{{Key: "id", Value: "art-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticlesController_GetArticleByID_NotFound(t *testing.T) {
	ctrl := newArticlesController(&mockArticlesRepoCtrl{byID: map[string]*database.Article{}})
	w := performRequest(ctrl.GetArticleByID, "GET", "/articles/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestArticlesController_GetArticleByID_MissingParam(t *testing.T) {
	ctrl := newArticlesController(&mockArticlesRepoCtrl{})
	w := performRequest(ctrl.GetArticleByID, "GET", "/articles/", nil, gin.Params{{Key: "id", Value: ""}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticlesController_CreateArticle_Success(t *testing.T) {
	ctrl := newArticlesController(&mockArticlesRepoCtrl{})
	body := requests.Article{SKU: "NEW-001", Name: "New Article", Presentation: "unit"}
	w := performRequest(ctrl.CreateArticle, "POST", "/articles", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestArticlesController_CreateArticle_InvalidJSON(t *testing.T) {
	ctrl := newArticlesController(&mockArticlesRepoCtrl{})
	// Send empty body — missing required fields
	w := performRequest(ctrl.CreateArticle, "POST", "/articles", nil, nil)
	// ShouldBindJSON with empty body fails validation
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestArticlesController_CreateArticle_Conflict(t *testing.T) {
	repo := &mockArticlesRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "SKU already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newArticlesController(repo)
	body := requests.Article{SKU: "DUP", Name: "Duplicate", Presentation: "unit"}
	w := performRequest(ctrl.CreateArticle, "POST", "/articles", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestArticlesController_DeleteArticle_Success(t *testing.T) {
	repo := &mockArticlesRepoCtrl{
		byID: map[string]*database.Article{"art-1": {ID: "art-1", SKU: "SKU1", Name: "Art1", Presentation: "unit"}},
	}
	ctrl := newArticlesController(repo)
	w := performRequest(ctrl.DeleteArticle, "DELETE", "/articles/art-1", nil, gin.Params{{Key: "id", Value: "art-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestArticlesController_DeleteArticle_Error(t *testing.T) {
	repo := &mockArticlesRepoCtrl{
		byID: map[string]*database.Article{"art-1": {ID: "art-1", SKU: "SKU1", Name: "Art1", Presentation: "unit"}},
		deleteErr: &responses.InternalResponse{
			Message:    "db error",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newArticlesController(repo)
	w := performRequest(ctrl.DeleteArticle, "DELETE", "/articles/art-1", nil, gin.Params{{Key: "id", Value: "art-1"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestArticlesController_ExportArticlesToExcel(t *testing.T) {
	ctrl := newArticlesController(&mockArticlesRepoCtrl{})
	w := performRequest(ctrl.ExportArticlesToExcel, "GET", "/articles/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}
