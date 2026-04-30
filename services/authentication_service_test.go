package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthRepo is an in-memory fake for unit testing AuthenticationService.
type mockAuthRepo struct {
	loginResp         *responses.LoginResponse
	loginErr          *responses.InternalResponse
	requestResetResp  *responses.InternalResponse
	resetPasswordResp *responses.InternalResponse
}

func (m *mockAuthRepo) Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse) {
	return m.loginResp, m.loginErr
}

func (m *mockAuthRepo) RequestPasswordReset(_ context.Context, _ string, _ string) *responses.InternalResponse {
	return m.requestResetResp
}

func (m *mockAuthRepo) ResetPassword(_ context.Context, _, _ string) *responses.InternalResponse {
	return m.resetPasswordResp
}

// mockRolesRepo is an in-memory fake for the RolesRepository used in authentication.
type mockRolesRepo struct {
	permissions json.RawMessage
	permErr     error
	roleEntry   *ports.RoleEntry
	roleErr     error
}

func (m *mockRolesRepo) GetRolePermissions(ctx context.Context, roleID string) ([]byte, error) {
	return m.permissions, m.permErr
}

func (m *mockRolesRepo) List(ctx context.Context) ([]ports.RoleEntry, error) {
	return nil, nil
}

func (m *mockRolesRepo) GetByID(ctx context.Context, roleID string) (*ports.RoleEntry, error) {
	return m.roleEntry, m.roleErr
}

func (m *mockRolesRepo) UpdatePermissions(ctx context.Context, roleID string, permissions json.RawMessage) error {
	return nil
}

// --- Tests ---

func TestAuthenticationService_Login_Success(t *testing.T) {
	authRepo := &mockAuthRepo{
		loginResp: &responses.LoginResponse{
			Name:     "Alice",
			LastName: "Smith",
			Email:    "alice@example.com",
			Token:    "jwt-token",
			Role:     "role-1",
		},
	}
	svc := NewAuthenticationService(authRepo, nil)
	resp, errResp := svc.Login(requests.Login{Email: "alice@example.com", Password: "secret"})
	require.Nil(t, errResp)
	require.NotNil(t, resp)
	assert.Equal(t, "alice@example.com", resp.Email)
	assert.Equal(t, "jwt-token", resp.Token)
	// No roles repo: role stays as the raw roleID returned by the auth repo
	assert.Equal(t, "role-1", resp.Role)
}

func TestAuthenticationService_Login_InvalidCredentials(t *testing.T) {
	authRepo := &mockAuthRepo{
		loginErr: &responses.InternalResponse{
			Message:    "credenciales inválidas",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	svc := NewAuthenticationService(authRepo, nil)
	resp, errResp := svc.Login(requests.Login{Email: "bad@example.com", Password: "wrong"})
	require.NotNil(t, errResp)
	assert.Nil(t, resp)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
}

func TestAuthenticationService_Login_WithRolesRepo_AttachesPermissionsAndRoleName(t *testing.T) {
	perms := json.RawMessage(`["read","write"]`)
	authRepo := &mockAuthRepo{
		loginResp: &responses.LoginResponse{
			Name:     "Bob",
			LastName: "Jones",
			Email:    "bob@example.com",
			Token:    "jwt-bob",
			Role:     "role-admin",
		},
	}
	rolesRepo := &mockRolesRepo{
		permissions: perms,
		permErr:     nil,
		roleEntry: &ports.RoleEntry{
			ID:   "role-admin",
			Name: "Admin",
		},
	}
	svc := NewAuthenticationService(authRepo, rolesRepo)
	resp, errResp := svc.Login(requests.Login{Email: "bob@example.com", Password: "secret"})
	require.Nil(t, errResp)
	require.NotNil(t, resp)
	assert.Equal(t, "Admin", resp.Role)
	assert.Equal(t, json.RawMessage(`["read","write"]`), resp.Permissions)
}

func TestAuthenticationService_Login_WithRolesRepo_PermError_SkipsPermissions(t *testing.T) {
	authRepo := &mockAuthRepo{
		loginResp: &responses.LoginResponse{
			Name:  "Carol",
			Email: "carol@example.com",
			Token: "jwt-carol",
			Role:  "role-viewer",
		},
	}
	rolesRepo := &mockRolesRepo{
		permissions: nil,
		permErr:     errors.New("permissions fetch failed"),
		roleEntry: &ports.RoleEntry{
			ID:   "role-viewer",
			Name: "Viewer",
		},
	}
	svc := NewAuthenticationService(authRepo, rolesRepo)
	resp, errResp := svc.Login(requests.Login{Email: "carol@example.com", Password: "secret"})
	require.Nil(t, errResp)
	require.NotNil(t, resp)
	// permissions fetch errored → not attached
	assert.Nil(t, resp.Permissions)
	// role name is still resolved
	assert.Equal(t, "Viewer", resp.Role)
}

func TestAuthenticationService_Login_NilResponse_ReturnsError(t *testing.T) {
	authRepo := &mockAuthRepo{
		loginResp: nil,
		loginErr: &responses.InternalResponse{
			Message:    "usuario no encontrado",
			Handled:    true,
			StatusCode: responses.StatusNotFound,
		},
	}
	svc := NewAuthenticationService(authRepo, nil)
	resp, errResp := svc.Login(requests.Login{Email: "ghost@example.com", Password: "x"})
	require.NotNil(t, errResp)
	assert.Nil(t, resp)
	assert.Equal(t, responses.StatusNotFound, errResp.StatusCode)
}

func TestAuthenticationService_Login_WithRolesRepo_RoleHasNoID_SkipsRoleLookup(t *testing.T) {
	authRepo := &mockAuthRepo{
		loginResp: &responses.LoginResponse{
			Name:  "Dave",
			Email: "dave@example.com",
			Token: "jwt-dave",
			Role:  "", // empty role ID
		},
	}
	rolesRepo := &mockRolesRepo{
		roleEntry: &ports.RoleEntry{ID: "should-not-be-called", Name: "Should Not Appear"},
	}
	svc := NewAuthenticationService(authRepo, rolesRepo)
	resp, errResp := svc.Login(requests.Login{Email: "dave@example.com", Password: "secret"})
	require.Nil(t, errResp)
	require.NotNil(t, resp)
	// Role was empty so the roles repo branch is skipped; role stays empty
	assert.Equal(t, "", resp.Role)
}
