package controllers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingUsersRepo records the args passed to CreateUser/UpdateUser so the tests
// can assert the mobile controller maps the trimmed DTO → requests.User correctly
// (name split, tenant scope, role pass-through) before delegating to the SAME
// service the web path uses.
type capturingUsersRepo struct {
	users          []database.User
	createErr      *responses.InternalResponse
	updateErr      *responses.InternalResponse
	lastTenantID   string
	lastCreateUser *requests.User
	lastUpdateID   string
	lastUpdateData map[string]interface{}
}

func (m *capturingUsersRepo) GetAllUsers() ([]database.User, *responses.InternalResponse) {
	return m.users, nil
}
func (m *capturingUsersRepo) GetUsersByTenant(tenantID string) ([]database.User, *responses.InternalResponse) {
	m.lastTenantID = tenantID
	return m.users, nil
}
func (m *capturingUsersRepo) GetUserByID(id string) (*database.User, *responses.InternalResponse) {
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}
func (m *capturingUsersRepo) CreateUser(tenantID string, user *requests.User) *responses.InternalResponse {
	m.lastTenantID = tenantID
	m.lastCreateUser = user
	return m.createErr
}
func (m *capturingUsersRepo) UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse {
	m.lastUpdateID = id
	m.lastUpdateData = data
	return m.updateErr
}
func (m *capturingUsersRepo) DeleteUser(id string) *responses.InternalResponse { return nil }
func (m *capturingUsersRepo) ImportUsersFromExcel(_ string, _ []byte) ([]string, []*responses.InternalResponse) {
	return nil, nil
}
func (m *capturingUsersRepo) ExportUsersToExcel() ([]byte, *responses.InternalResponse) {
	return nil, nil
}
func (m *capturingUsersRepo) UpdateUserPassword(id string, newPassword string) *responses.InternalResponse {
	return nil
}
func (m *capturingUsersRepo) GenerateImportTemplate(language string) ([]byte, error) {
	return nil, nil
}

func newTestMobileUsersController(repo *capturingUsersRepo, roles ports.RolesRepository) *MobileUsersController {
	svc := services.NewUserService(repo)
	cfg := configuration.Config{JWTSecret: testJWTSecret, TenantID: "tenant-test"}
	return NewMobileUsersController(svc, roles, cfg)
}

// ─── GET /api/mobile/users ────────────────────────────────────────────────────

func TestMobileUsers_List_TrimsPasswordAndTenant(t *testing.T) {
	pw := "should-never-leak"
	repo := &capturingUsersRepo{users: []database.User{
		{ID: "u1", Name: "Ana García", Email: "ana@co.com", RoleID: "r1", IsActive: true, TenantID: "secret-uuid", Password: &pw, Role: &database.Role{ID: "r1", Name: "Admin"}},
	}}
	ctrl := newTestMobileUsersController(repo, nil)

	w := performRequest(ctrl.ListUsers, "GET", "/api/mobile/users", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.NotContains(t, body, "should-never-leak", "password must never leak")
	assert.NotContains(t, body, "secret-uuid", "tenant UUID must never leak")

	var env responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.True(t, env.Result.Success)
	arr, ok := env.Data.([]interface{})
	require.True(t, ok)
	require.Len(t, arr, 1)
	row := arr[0].(map[string]interface{})
	assert.Equal(t, "Ana García", row["name"])
	assert.Equal(t, "Admin", row["role_name"])
	assert.Equal(t, true, row["is_active"])
}

// ─── POST /api/mobile/users ───────────────────────────────────────────────────

func TestMobileUsers_Create_Success_MapsFieldsAndTenant(t *testing.T) {
	repo := &capturingUsersRepo{}
	ctrl := newTestMobileUsersController(repo, nil)

	body := MobileCreateUserRequest{Name: "Ana García", Email: "ana@co.com", Password: "Passw0rd!", RoleID: "Operator"}
	w := performRequest(ctrl.CreateUser, "POST", "/api/mobile/users", body, nil)
	require.Equal(t, http.StatusCreated, w.Code)

	require.NotNil(t, repo.lastCreateUser)
	assert.Equal(t, "tenant-test", repo.lastTenantID, "tenant must fall back to Config when no JWT ctx")
	assert.Equal(t, "ana@co.com", repo.lastCreateUser.Email)
	assert.Equal(t, "Ana", repo.lastCreateUser.FirstName)
	assert.Equal(t, "García", repo.lastCreateUser.LastName)
	assert.Equal(t, "Operator", repo.lastCreateUser.RoleID)
	require.NotNil(t, repo.lastCreateUser.Password)
	assert.Equal(t, "Passw0rd!", *repo.lastCreateUser.Password, "controller passes plaintext; service hashes via tools.Encrypt")
}

func TestMobileUsers_Create_ValidationError(t *testing.T) {
	repo := &capturingUsersRepo{}
	ctrl := newTestMobileUsersController(repo, nil)

	// missing password + bad email
	body := MobileCreateUserRequest{Name: "X", Email: "not-an-email", RoleID: "Operator"}
	w := performRequest(ctrl.CreateUser, "POST", "/api/mobile/users", body, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, repo.lastCreateUser, "service must not be called on validation failure")
}

func TestMobileUsers_Create_DuplicateEmailConflict(t *testing.T) {
	repo := &capturingUsersRepo{createErr: &responses.InternalResponse{Message: "El correo electrónico ya existe", Handled: true, StatusCode: responses.StatusConflict}}
	ctrl := newTestMobileUsersController(repo, nil)

	body := MobileCreateUserRequest{Name: "Ana", Email: "dup@co.com", Password: "Passw0rd!", RoleID: "Operator"}
	w := performRequest(ctrl.CreateUser, "POST", "/api/mobile/users", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ─── PATCH /api/mobile/users/:id ──────────────────────────────────────────────

func TestMobileUsers_Update_Deactivate(t *testing.T) {
	repo := &capturingUsersRepo{}
	ctrl := newTestMobileUsersController(repo, nil)

	active := false
	body := MobileUpdateUserRequest{IsActive: &active}
	w := performRequest(ctrl.UpdateUser, "PATCH", "/api/mobile/users/u1", body, gin.Params{{Key: "id", Value: "u1"}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "u1", repo.lastUpdateID)
	require.Contains(t, repo.lastUpdateData, "is_active")
	assert.Equal(t, false, repo.lastUpdateData["is_active"])
	assert.NotContains(t, repo.lastUpdateData, "role_id")
}

func TestMobileUsers_Update_ChangeRole(t *testing.T) {
	repo := &capturingUsersRepo{}
	ctrl := newTestMobileUsersController(repo, nil)

	role := "Admin"
	body := MobileUpdateUserRequest{RoleID: &role}
	w := performRequest(ctrl.UpdateUser, "PATCH", "/api/mobile/users/u1", body, gin.Params{{Key: "id", Value: "u1"}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Admin", repo.lastUpdateData["role_id"])
}

func TestMobileUsers_Update_EmptyBodyRejected(t *testing.T) {
	repo := &capturingUsersRepo{}
	ctrl := newTestMobileUsersController(repo, nil)

	body := MobileUpdateUserRequest{}
	w := performRequest(ctrl.UpdateUser, "PATCH", "/api/mobile/users/u1", body, gin.Params{{Key: "id", Value: "u1"}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, repo.lastUpdateData, "service must not be called when nothing to update")
}

// ─── GET /api/mobile/roles ────────────────────────────────────────────────────

func TestMobileUsers_ListRoles(t *testing.T) {
	roles := &mockRolesRepo{roles: []ports.RoleEntry{{ID: "r1", Name: "Admin"}, {ID: "r2", Name: "Operator"}}}
	ctrl := newTestMobileUsersController(&capturingUsersRepo{}, roles)

	w := performRequest(ctrl.ListRoles, "GET", "/api/mobile/roles", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)

	var env responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	arr, ok := env.Data.([]interface{})
	require.True(t, ok)
	require.Len(t, arr, 2)
	assert.Equal(t, "Admin", arr[0].(map[string]interface{})["name"])
	// roles DTO must not carry the permissions blob
	assert.NotContains(t, w.Body.String(), "permissions")
}

func TestMobileUsers_ListRoles_NilRepoGraceful(t *testing.T) {
	ctrl := newTestMobileUsersController(&capturingUsersRepo{}, nil)
	w := performRequest(ctrl.ListRoles, "GET", "/api/mobile/roles", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}
