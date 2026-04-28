package services

import (
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUsersRepo is an in-memory fake for unit testing UserService.
type mockUsersRepo struct {
	users         []database.User
	byID          map[string]*database.User
	createErr     *responses.InternalResponse
	updateErr     *responses.InternalResponse
	deleteErr     *responses.InternalResponse
	passwordErr   *responses.InternalResponse
	exportBytes   []byte
	exportErr     *responses.InternalResponse
	importedIDs   []string
	importErrs    []*responses.InternalResponse
	templateBytes []byte
	templateErr   error
}

func (m *mockUsersRepo) GetAllUsers() ([]database.User, *responses.InternalResponse) {
	return m.users, nil
}

func (m *mockUsersRepo) GetUserByID(id string) (*database.User, *responses.InternalResponse) {
	if m.byID != nil {
		if u, ok := m.byID[id]; ok {
			return u, nil
		}
	}
	return nil, &responses.InternalResponse{
		Message:    "usuario no encontrado",
		Handled:    true,
		StatusCode: responses.StatusNotFound,
	}
}

func (m *mockUsersRepo) CreateUser(_ string, user *requests.User) *responses.InternalResponse {
	if m.createErr != nil {
		return m.createErr
	}
	m.users = append(m.users, database.User{
		ID:        "new-user-id",
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		RoleID:    user.RoleID,
	})
	return nil
}

func (m *mockUsersRepo) UpdateUser(id string, data map[string]interface{}) *responses.InternalResponse {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *mockUsersRepo) DeleteUser(id string) *responses.InternalResponse {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *mockUsersRepo) ImportUsersFromExcel(_ string, fileBytes []byte) ([]string, []*responses.InternalResponse) {
	return m.importedIDs, m.importErrs
}

func (m *mockUsersRepo) ExportUsersToExcel() ([]byte, *responses.InternalResponse) {
	return m.exportBytes, m.exportErr
}

func (m *mockUsersRepo) UpdateUserPassword(id string, newPassword string) *responses.InternalResponse {
	if m.passwordErr != nil {
		return m.passwordErr
	}
	return nil
}

func (m *mockUsersRepo) GenerateImportTemplate(language string) ([]byte, error) {
	return m.templateBytes, m.templateErr
}

// --- Tests ---

func TestUserService_GetAllUsers(t *testing.T) {
	repo := &mockUsersRepo{
		users: []database.User{
			{ID: "1", Email: "alice@example.com", FirstName: "Alice", LastName: "Smith"},
			{ID: "2", Email: "bob@example.com", FirstName: "Bob", LastName: "Jones"},
		},
	}
	svc := NewUserService(repo)
	list, errResp := svc.GetAllUsers()
	require.Nil(t, errResp)
	require.Len(t, list, 2)
	assert.Equal(t, "alice@example.com", list[0].Email)
	assert.Equal(t, "bob@example.com", list[1].Email)
}

func TestUserService_GetAllUsers_Empty(t *testing.T) {
	repo := &mockUsersRepo{users: []database.User{}}
	svc := NewUserService(repo)
	list, errResp := svc.GetAllUsers()
	require.Nil(t, errResp)
	assert.Empty(t, list)
}

func TestUserService_GetUserByID_Found(t *testing.T) {
	repo := &mockUsersRepo{
		byID: map[string]*database.User{
			"1": {ID: "1", Email: "alice@example.com", FirstName: "Alice"},
		},
	}
	svc := NewUserService(repo)
	user, errResp := svc.GetUserByID("1")
	require.Nil(t, errResp)
	require.NotNil(t, user)
	assert.Equal(t, "alice@example.com", user.Email)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	repo := &mockUsersRepo{byID: map[string]*database.User{}}
	svc := NewUserService(repo)
	user, errResp := svc.GetUserByID("99")
	require.NotNil(t, errResp)
	assert.Nil(t, user)
	assert.True(t, errResp.Handled)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestUserService_CreateUser_Success(t *testing.T) {
	repo := &mockUsersRepo{users: []database.User{}}
	svc := NewUserService(repo)
	req := &requests.User{
		Email:     "newuser@example.com",
		FirstName: "New",
		LastName:  "User",
		RoleID:    "role-1",
	}
	errResp := svc.CreateUser("tenant-1", req)
	require.Nil(t, errResp)
	require.Len(t, repo.users, 1)
	assert.Equal(t, "newuser@example.com", repo.users[0].Email)
}

func TestUserService_CreateUser_Conflict(t *testing.T) {
	repo := &mockUsersRepo{
		createErr: &responses.InternalResponse{
			Message:    "ya existe un usuario con ese email",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewUserService(repo)
	req := &requests.User{Email: "dup@example.com", FirstName: "Dup", LastName: "User", RoleID: "role-1"}
	errResp := svc.CreateUser("tenant-1", req)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusConflict, errResp.StatusCode)
}

func TestUserService_UpdateUser_Success(t *testing.T) {
	repo := &mockUsersRepo{}
	svc := NewUserService(repo)
	errResp := svc.UpdateUser("1", map[string]interface{}{"first_name": "Updated"})
	require.Nil(t, errResp)
}

func TestUserService_UpdateUser_NotFound(t *testing.T) {
	repo := &mockUsersRepo{
		updateErr: &responses.InternalResponse{
			Message:    "usuario no encontrado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewUserService(repo)
	errResp := svc.UpdateUser("99", map[string]interface{}{"first_name": "X"})
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestUserService_DeleteUser_Success(t *testing.T) {
	repo := &mockUsersRepo{}
	svc := NewUserService(repo)
	errResp := svc.DeleteUser("1")
	require.Nil(t, errResp)
}

func TestUserService_DeleteUser_Error(t *testing.T) {
	repo := &mockUsersRepo{
		deleteErr: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "error al eliminar usuario",
			Handled: false,
		},
	}
	svc := NewUserService(repo)
	errResp := svc.DeleteUser("1")
	require.NotNil(t, errResp)
	assert.False(t, errResp.Handled)
}

func TestUserService_UpdateUserPassword_Success(t *testing.T) {
	repo := &mockUsersRepo{}
	svc := NewUserService(repo)
	errResp := svc.UpdateUserPassword("1", "newpassword123")
	require.Nil(t, errResp)
}

func TestUserService_UpdateUserPassword_NotFound(t *testing.T) {
	repo := &mockUsersRepo{
		passwordErr: &responses.InternalResponse{
			Message:    "usuario no encontrado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewUserService(repo)
	errResp := svc.UpdateUserPassword("99", "pass")
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestUserService_ExportUsersToExcel_Success(t *testing.T) {
	repo := &mockUsersRepo{exportBytes: []byte("excel-bytes")}
	svc := NewUserService(repo)
	data, errResp := svc.ExportUsersToExcel()
	require.Nil(t, errResp)
	assert.Equal(t, []byte("excel-bytes"), data)
}

func TestUserService_ExportUsersToExcel_Error(t *testing.T) {
	repo := &mockUsersRepo{
		exportErr: &responses.InternalResponse{
			Error:   errors.New("export failed"),
			Message: "error al exportar",
			Handled: false,
		},
	}
	svc := NewUserService(repo)
	data, errResp := svc.ExportUsersToExcel()
	require.NotNil(t, errResp)
	assert.Nil(t, data)
}

func TestUserService_ImportUsersFromExcel_Success(t *testing.T) {
	repo := &mockUsersRepo{
		importedIDs: []string{"user-1", "user-2"},
		importErrs:  nil,
	}
	svc := NewUserService(repo)
	ids, errs := svc.ImportUsersFromExcel("tenant-1", []byte("some-excel"))
	assert.Len(t, ids, 2)
	assert.Nil(t, errs)
}

func TestUserService_GenerateImportTemplate_Success(t *testing.T) {
	repo := &mockUsersRepo{templateBytes: []byte("template-bytes")}
	svc := NewUserService(repo)
	data, err := svc.GenerateImportTemplate("en")
	require.NoError(t, err)
	assert.Equal(t, []byte("template-bytes"), data)
}

func TestUserService_GenerateImportTemplate_Error(t *testing.T) {
	repo := &mockUsersRepo{templateErr: errors.New("template generation failed")}
	svc := NewUserService(repo)
	data, err := svc.GenerateImportTemplate("en")
	require.Error(t, err)
	assert.Nil(t, data)
}
