package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock roles repo ──────────────────────────────────────────────────────────

type mockRolesRepo struct {
	roles    []ports.RoleEntry
	byID     map[string]*ports.RoleEntry
	listErr  error
	getErr   error
	updateErr error
}

func (m *mockRolesRepo) GetRolePermissions(_ context.Context, roleID string) ([]byte, error) {
	if m.byID != nil {
		if r, ok := m.byID[roleID]; ok {
			return r.Permissions, nil
		}
	}
	return nil, nil
}

func (m *mockRolesRepo) List(_ context.Context) ([]ports.RoleEntry, error) {
	return m.roles, m.listErr
}

func (m *mockRolesRepo) GetByID(_ context.Context, id string) (*ports.RoleEntry, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.byID != nil {
		if r, ok := m.byID[id]; ok {
			return r, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockRolesRepo) UpdatePermissions(_ context.Context, roleID string, permissions json.RawMessage) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.byID != nil {
		if r, ok := m.byID[roleID]; ok {
			r.Permissions = permissions
		}
	}
	return nil
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestRolesController_ListRoles_Success(t *testing.T) {
	repo := &mockRolesRepo{
		roles: []ports.RoleEntry{{ID: "r1", Name: "admin"}},
	}
	ctrl := NewRolesController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/roles", nil)
	c.Request = req
	ctrl.ListRoles(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRolesController_ListRoles_NilRepo(t *testing.T) {
	ctrl := NewRolesController(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/roles", nil)
	c.Request = req
	ctrl.ListRoles(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRolesController_ListRoles_Error(t *testing.T) {
	repo := &mockRolesRepo{listErr: errors.New("db error")}
	ctrl := NewRolesController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/roles", nil)
	c.Request = req
	ctrl.ListRoles(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRolesController_GetRoleByID_Found(t *testing.T) {
	repo := &mockRolesRepo{
		byID: map[string]*ports.RoleEntry{"r1": {ID: "r1", Name: "admin"}},
	}
	ctrl := NewRolesController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/roles/r1", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "r1"}}
	ctrl.GetRoleByID(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRolesController_GetRoleByID_NotFound(t *testing.T) {
	repo := &mockRolesRepo{}
	ctrl := NewRolesController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/roles/bad", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	ctrl.GetRoleByID(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRolesController_GetRoleByID_MissingParam(t *testing.T) {
	ctrl := NewRolesController(&mockRolesRepo{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/roles/", nil)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: ""}}
	ctrl.GetRoleByID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRolesController_UpdateRolePermissions_Success(t *testing.T) {
	perms := json.RawMessage(`{"articles":{"read":true}}`)
	repo := &mockRolesRepo{
		byID: map[string]*ports.RoleEntry{"r1": {ID: "r1", Name: "admin", Permissions: perms}},
	}
	ctrl := NewRolesController(repo)

	body := map[string]interface{}{"permissions": map[string]interface{}{"articles": map[string]bool{"read": true}}}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("PUT", "/roles/r1", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "r1"}}
	ctrl.UpdateRolePermissions(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRolesController_UpdateRolePermissions_EmptyPerms(t *testing.T) {
	ctrl := NewRolesController(&mockRolesRepo{byID: map[string]*ports.RoleEntry{}})

	// Omitting the "permissions" key means json.RawMessage is nil/empty
	b := []byte(`{}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("PUT", "/roles/r1", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: "r1"}}
	ctrl.UpdateRolePermissions(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
