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
)

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockCategoriesRepo struct {
	categories []database.Category
	byID       map[string]*database.Category
	createErr  *responses.InternalResponse
}

func (m *mockCategoriesRepo) Create(_ string, data *requests.CreateCategoryRequest) (*database.Category, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	c := &database.Category{ID: "cat-id", Name: data.Name, ParentID: data.ParentID, IsActive: true}
	return c, nil
}

func (m *mockCategoriesRepo) GetByID(id string) (*database.Category, *responses.InternalResponse) {
	if m.byID != nil {
		if c, ok := m.byID[id]; ok {
			return c, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockCategoriesRepo) ListByTenant(_ string) ([]database.Category, *responses.InternalResponse) {
	return m.categories, nil
}

func (m *mockCategoriesRepo) Update(id string, data *requests.UpdateCategoryRequest) (*database.Category, *responses.InternalResponse) {
	if m.byID != nil {
		if c, ok := m.byID[id]; ok {
			c.Name = data.Name
			c.ParentID = data.ParentID
			return c, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockCategoriesRepo) SoftDelete(id string) *responses.InternalResponse {
	if m.byID != nil {
		if _, ok := m.byID[id]; ok {
			return nil
		}
	}
	return &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func newCategoriesController(repo *mockCategoriesRepo) *CategoriesController {
	svc := services.NewCategoriesService(repo)
	return NewCategoriesController(*svc, "00000000-0000-0000-0000-000000000001")
}

func newCategoriesRouter(ctrl *CategoriesController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/categories", ctrl.List)
	r.GET("/categories/tree", ctrl.GetTree)
	r.GET("/categories/:id", ctrl.GetByID)
	r.POST("/categories", ctrl.Create)
	r.PATCH("/categories/:id", ctrl.Update)
	r.DELETE("/categories/:id", ctrl.SoftDelete)
	return r
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestCategoriesController_Create_Root(t *testing.T) {
	repo := &mockCategoriesRepo{}
	ctrl := newCategoriesController(repo)
	r := newCategoriesRouter(ctrl)

	body := requests.CreateCategoryRequest{Name: "Electrónicos"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCategoriesController_Create_Child(t *testing.T) {
	parentID := "root-id"
	parent := &database.Category{ID: parentID, Name: "Electrónicos", IsActive: true}
	repo := &mockCategoriesRepo{
		byID: map[string]*database.Category{parentID: parent},
	}
	ctrl := newCategoriesController(repo)
	r := newCategoriesRouter(ctrl)

	body := requests.CreateCategoryRequest{Name: "Celulares", ParentID: &parentID}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCategoriesController_Create_InvalidParent(t *testing.T) {
	repo := &mockCategoriesRepo{}
	ctrl := newCategoriesController(repo)
	r := newCategoriesRouter(ctrl)

	pid := "nonexistent"
	body := requests.CreateCategoryRequest{Name: "Sub", ParentID: &pid}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCategoriesController_Create_TooDeep(t *testing.T) {
	rootID := "root-id"
	childID := "child-id"
	root := &database.Category{ID: rootID, Name: "Root", IsActive: true}
	child := &database.Category{ID: childID, Name: "Child", ParentID: &rootID, IsActive: true}
	repo := &mockCategoriesRepo{
		byID: map[string]*database.Category{rootID: root, childID: child},
	}
	ctrl := newCategoriesController(repo)
	r := newCategoriesRouter(ctrl)

	// Try to create a grandchild (3rd level) — should fail
	body := requests.CreateCategoryRequest{Name: "Grandchild", ParentID: &childID}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/categories", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCategoriesController_GetTree_Structure(t *testing.T) {
	rootID := "root-id"
	repo := &mockCategoriesRepo{
		categories: []database.Category{
			{ID: rootID, Name: "Root", IsActive: true},
			{ID: "child-1", Name: "Child 1", ParentID: &rootID, IsActive: true},
			{ID: "child-2", Name: "Child 2", ParentID: &rootID, IsActive: true},
		},
	}
	ctrl := newCategoriesController(repo)
	r := newCategoriesRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/categories/tree", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1) // one root
	root := data[0].(map[string]interface{})
	children := root["children"].([]interface{})
	assert.Len(t, children, 2)
}

func TestCategoriesController_Update_CycleDetected(t *testing.T) {
	// parent tries to become a child of its own child
	parentID := "parent-id"
	childID := "child-id"
	parent := &database.Category{ID: parentID, Name: "Parent", IsActive: true}
	child := &database.Category{ID: childID, Name: "Child", ParentID: &parentID, IsActive: true}
	repo := &mockCategoriesRepo{
		byID:       map[string]*database.Category{parentID: parent, childID: child},
		categories: []database.Category{*parent, *child},
	}
	ctrl := newCategoriesController(repo)
	r := newCategoriesRouter(ctrl)

	body := requests.UpdateCategoryRequest{Name: "Parent", ParentID: &childID}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/categories/"+parentID, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
