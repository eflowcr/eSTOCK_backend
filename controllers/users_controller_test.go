package controllers

import (
	"net/http"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock repo ───────────────────────────────────────────────────────────────

type mockUsersRepoCtrl struct {
	users     []database.User
	byID      map[string]*database.User
	createErr *responses.InternalResponse
	updateErr *responses.InternalResponse
	deleteErr *responses.InternalResponse
	exportErr *responses.InternalResponse
}

func (m *mockUsersRepoCtrl) GetAllUsers() ([]database.User, *responses.InternalResponse) {
	return m.users, nil
}

func (m *mockUsersRepoCtrl) GetUserByID(id string) (*database.User, *responses.InternalResponse) {
	if m.byID != nil {
		if u, ok := m.byID[id]; ok {
			return u, nil
		}
	}
	return nil, &responses.InternalResponse{Message: "not found", Handled: true, StatusCode: responses.StatusNotFound}
}

func (m *mockUsersRepoCtrl) CreateUser(user *requests.User) *responses.InternalResponse {
	return m.createErr
}

func (m *mockUsersRepoCtrl) UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse {
	return m.updateErr
}

func (m *mockUsersRepoCtrl) DeleteUser(id string) *responses.InternalResponse {
	return m.deleteErr
}

func (m *mockUsersRepoCtrl) ImportUsersFromExcel(fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return []string{"user1"}, nil
}

func (m *mockUsersRepoCtrl) ExportUsersToExcel() ([]byte, *responses.InternalResponse) {
	return []byte("xlsx"), m.exportErr
}

func (m *mockUsersRepoCtrl) UpdateUserPassword(id string, newPassword string) *responses.InternalResponse {
	return nil
}

func (m *mockUsersRepoCtrl) GenerateImportTemplate(language string) ([]byte, error) {
	return []byte("tpl"), nil
}

// ─── helper ──────────────────────────────────────────────────────────────────

func newUsersController(repo *mockUsersRepoCtrl) *UserController {
	svc := services.NewUserService(repo)
	return NewUserController(*svc)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestUsersController_GetAllUsers_Empty(t *testing.T) {
	ctrl := newUsersController(&mockUsersRepoCtrl{users: []database.User{}})
	w := performRequest(ctrl.GetAllUsers, "GET", "/users", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUsersController_GetAllUsers_WithData(t *testing.T) {
	repo := &mockUsersRepoCtrl{
		users: []database.User{{ID: "u-1", Name: "John", Email: "john@example.com", RoleID: "role-1"}},
	}
	ctrl := newUsersController(repo)
	w := performRequest(ctrl.GetAllUsers, "GET", "/users", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUsersController_GetUserByID_Found(t *testing.T) {
	repo := &mockUsersRepoCtrl{
		byID: map[string]*database.User{
			"u-1": {ID: "u-1", Name: "John", Email: "john@example.com", RoleID: "role-1"},
		},
	}
	ctrl := newUsersController(repo)
	w := performRequest(ctrl.GetUserByID, "GET", "/users/u-1", nil, gin.Params{{Key: "id", Value: "u-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUsersController_GetUserByID_NotFound(t *testing.T) {
	ctrl := newUsersController(&mockUsersRepoCtrl{byID: map[string]*database.User{}})
	w := performRequest(ctrl.GetUserByID, "GET", "/users/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUsersController_CreateUser_Success(t *testing.T) {
	ctrl := newUsersController(&mockUsersRepoCtrl{})
	pw := "secret123"
	body := requests.User{
		Email:     "new@example.com",
		FirstName: "New",
		LastName:  "User",
		Password:  &pw,
		RoleID:    "role-1",
	}
	w := performRequest(ctrl.CreateUser, "POST", "/users", body, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestUsersController_CreateUser_InvalidJSON(t *testing.T) {
	ctrl := newUsersController(&mockUsersRepoCtrl{})
	w := performRequest(ctrl.CreateUser, "POST", "/users", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUsersController_CreateUser_Conflict(t *testing.T) {
	repo := &mockUsersRepoCtrl{
		createErr: &responses.InternalResponse{
			Message:    "email already exists",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newUsersController(repo)
	pw := "secret123"
	body := requests.User{
		Email:     "dup@example.com",
		FirstName: "Dup",
		LastName:  "User",
		Password:  &pw,
		RoleID:    "role-1",
	}
	w := performRequest(ctrl.CreateUser, "POST", "/users", body, nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUsersController_UpdateUser_Success(t *testing.T) {
	ctrl := newUsersController(&mockUsersRepoCtrl{})
	body := map[string]interface{}{"first_name": "Updated"}
	w := performRequest(ctrl.UpdateUser, "PUT", "/users/u-1", body, gin.Params{{Key: "id", Value: "u-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUsersController_UpdateUser_NotFound(t *testing.T) {
	repo := &mockUsersRepoCtrl{
		updateErr: &responses.InternalResponse{
			Message:    "user not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newUsersController(repo)
	body := map[string]interface{}{"first_name": "Ghost"}
	w := performRequest(ctrl.UpdateUser, "PUT", "/users/99", body, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUsersController_DeleteUser_Success(t *testing.T) {
	ctrl := newUsersController(&mockUsersRepoCtrl{})
	w := performRequest(ctrl.DeleteUser, "DELETE", "/users/u-1", nil, gin.Params{{Key: "id", Value: "u-1"}})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUsersController_DeleteUser_NotFound(t *testing.T) {
	repo := &mockUsersRepoCtrl{
		deleteErr: &responses.InternalResponse{
			Message:    "user not found",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	ctrl := newUsersController(repo)
	w := performRequest(ctrl.DeleteUser, "DELETE", "/users/99", nil, gin.Params{{Key: "id", Value: "99"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUsersController_ExportUsersToExcel_Success(t *testing.T) {
	ctrl := newUsersController(&mockUsersRepoCtrl{})
	w := performRequest(ctrl.ExportUsersToExcel, "GET", "/users/export", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUsersController_ExportUsersToExcel_Error(t *testing.T) {
	repo := &mockUsersRepoCtrl{
		exportErr: &responses.InternalResponse{
			Message:    "export failed",
			StatusCode: responses.StatusInternalServerError,
		},
	}
	ctrl := newUsersController(repo)
	w := performRequest(ctrl.ExportUsersToExcel, "GET", "/users/export", nil, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
