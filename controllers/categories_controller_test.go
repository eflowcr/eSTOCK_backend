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

// ListByTenantFiltered satisfies the updated ports.CategoriesRepository interface (M8).
// The mock applies isActive and search filters in-memory to support C1 wiring tests.
func (m *mockCategoriesRepo) ListByTenantFiltered(_ string, isActive *bool, search *string, limit *int32, offset *int32) ([]database.Category, *responses.InternalResponse) {
	out := make([]database.Category, 0, len(m.categories))
	for _, c := range m.categories {
		if isActive != nil && c.IsActive != *isActive {
			continue
		}
		if search != nil && *search != "" {
			if !containsIgnoreCaseCat(c.Name, *search) {
				continue
			}
		}
		out = append(out, c)
	}
	// apply offset/limit
	if offset != nil && int(*offset) < len(out) {
		out = out[int(*offset):]
	} else if offset != nil {
		out = []database.Category{}
	}
	if limit != nil && int(*limit) < len(out) {
		out = out[:int(*limit)]
	}
	return out, nil
}

func containsIgnoreCaseCat(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		func() bool {
			sl, subl := []rune(s), []rune(substr)
			for i := 0; i <= len(sl)-len(subl); i++ {
				match := true
				for j := range subl {
					sc, subc := sl[i+j], subl[j]
					if sc >= 'A' && sc <= 'Z' {
						sc += 32
					}
					if subc >= 'A' && subc <= 'Z' {
						subc += 32
					}
					if sc != subc {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
			return false
		}())
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

// TestCategoriesController_List_FilterParams verifies that C1 fix correctly wires
// is_active and search query params through to the SQL-filtered repository path (M8).
func TestCategoriesController_List_FilterParams(t *testing.T) {
	repo := &mockCategoriesRepo{
		categories: []database.Category{
			{ID: "c1", Name: "Electrónicos", IsActive: true},
			{ID: "c2", Name: "Ropa", IsActive: false},
			{ID: "c3", Name: "Electrodomésticos", IsActive: true},
		},
	}
	ctrl := newCategoriesController(repo)
	r := newCategoriesRouter(ctrl)

	t.Run("is_active=true returns only active", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/categories?is_active=true", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 2) // c1 + c3
	})

	t.Run("search filters by name substring", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/categories?search=Electr", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 2) // c1 + c3
	})

	t.Run("no params returns all categories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/categories", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		assert.Len(t, data, 3)
	})
}
