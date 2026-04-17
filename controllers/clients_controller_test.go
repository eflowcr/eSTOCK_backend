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

type mockClientsRepo struct {
	clients   []database.Client
	byID      map[string]*database.Client
	byCode    map[string]*database.Client
	createErr *responses.InternalResponse
}

func (m *mockClientsRepo) Create(_ string, data *requests.CreateClientRequest, _ *string) (*database.Client, *responses.InternalResponse) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	c := &database.Client{ID: "test-id", Type: data.Type, Code: data.Code, Name: data.Name, IsActive: true}
	return c, nil
}

func (m *mockClientsRepo) GetByID(id string) (*database.Client, *responses.InternalResponse) {
	if m.byID != nil {
		if c, ok := m.byID[id]; ok {
			return c, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockClientsRepo) GetByTenantAndCode(_, code string) (*database.Client, *responses.InternalResponse) {
	if m.byCode != nil {
		if c, ok := m.byCode[code]; ok {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockClientsRepo) ListByTenant(_ string) ([]database.Client, *responses.InternalResponse) {
	return m.clients, nil
}

func (m *mockClientsRepo) Update(id string, data *requests.UpdateClientRequest) (*database.Client, *responses.InternalResponse) {
	if m.byID != nil {
		if c, ok := m.byID[id]; ok {
			c.Name = data.Name
			return c, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockClientsRepo) SoftDelete(id string) *responses.InternalResponse {
	if m.byID != nil {
		if _, ok := m.byID[id]; ok {
			return nil
		}
	}
	return &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func newClientsController(repo *mockClientsRepo) *ClientsController {
	svc := services.NewClientsService(repo)
	return NewClientsController(*svc, "00000000-0000-0000-0000-000000000001")
}

func newClientsRouter(ctrl *ClientsController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/clients", ctrl.List)
	r.GET("/clients/:id", ctrl.GetByID)
	r.POST("/clients", ctrl.Create)
	r.PATCH("/clients/:id", ctrl.Update)
	r.DELETE("/clients/:id", ctrl.SoftDelete)
	return r
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestClientsController_List(t *testing.T) {
	repo := &mockClientsRepo{
		clients: []database.Client{
			{ID: "c1", Type: "supplier", Code: "S001", Name: "Proveedor A", IsActive: true},
		},
	}
	ctrl := newClientsController(repo)
	r := newClientsRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/clients", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientsController_Create_HappyPath(t *testing.T) {
	repo := &mockClientsRepo{}
	ctrl := newClientsController(repo)
	r := newClientsRouter(ctrl)

	body := requests.CreateClientRequest{Type: "supplier", Code: "S001", Name: "Proveedor A"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestClientsController_Create_InvalidType(t *testing.T) {
	repo := &mockClientsRepo{}
	ctrl := newClientsController(repo)
	r := newClientsRouter(ctrl)

	body := map[string]string{"type": "invalid", "code": "S001", "name": "Test"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClientsController_Create_CodeConflict(t *testing.T) {
	existing := &database.Client{ID: "existing", Code: "S001", Type: "supplier", Name: "Existing"}
	repo := &mockClientsRepo{
		byCode: map[string]*database.Client{"S001": existing},
	}
	ctrl := newClientsController(repo)
	r := newClientsRouter(ctrl)

	body := requests.CreateClientRequest{Type: "supplier", Code: "S001", Name: "New Supplier"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/clients", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestClientsController_GetByID_NotFound(t *testing.T) {
	repo := &mockClientsRepo{}
	ctrl := newClientsController(repo)
	r := newClientsRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/clients/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestClientsController_SoftDelete_HappyPath(t *testing.T) {
	c := &database.Client{ID: "c1", Code: "S001", Type: "supplier", Name: "A"}
	repo := &mockClientsRepo{byID: map[string]*database.Client{"c1": c}}
	ctrl := newClientsController(repo)
	r := newClientsRouter(ctrl)

	req := httptest.NewRequest(http.MethodDelete, "/clients/c1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClientsController_List_FilterByType(t *testing.T) {
	repo := &mockClientsRepo{
		clients: []database.Client{
			{ID: "c1", Type: "supplier", Code: "S001", Name: "Proveedor", IsActive: true},
			{ID: "c2", Type: "customer", Code: "C001", Name: "Cliente", IsActive: true},
		},
	}
	ctrl := newClientsController(repo)
	r := newClientsRouter(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/clients?type=supplier", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)
}
