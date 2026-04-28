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

// ─── mock signup repo ─────────────────────────────────────────────────────────

type mockSignupRepo struct {
	initiateResp *responses.InternalResponse
	verifyResp   *responses.SignupVerifiedResponse
	verifyErr    *responses.InternalResponse
}

func (m *mockSignupRepo) InitiateSignup(_ context.Context, _ requests.SignupRequest, _ string) *responses.InternalResponse {
	return m.initiateResp
}

func (m *mockSignupRepo) VerifySignup(_ context.Context, _ string) (*responses.SignupVerifiedResponse, *responses.InternalResponse) {
	return m.verifyResp, m.verifyErr
}

// ─── InitiateSignup ───────────────────────────────────────────────────────────

func TestSignupService_InitiateSignup_Success(t *testing.T) {
	repo := &mockSignupRepo{initiateResp: nil} // nil = success
	svc := NewSignupService(repo, nil)

	resp := svc.InitiateSignup(context.Background(), requests.SignupRequest{
		Email:         "admin@newco.com",
		CompanyName:   "New Company SA",
		TenantSlug:    "newcompany",
		AdminName:     "John Doe",
		AdminPassword: "supersecret123",
	}, "")

	assert.Nil(t, resp, "expected no error on success")
}

func TestSignupService_InitiateSignup_EmailConflict(t *testing.T) {
	repo := &mockSignupRepo{
		initiateResp: &responses.InternalResponse{
			Message:    "Ya existe una cuenta registrada con ese email",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	svc := NewSignupService(repo, nil)

	resp := svc.InitiateSignup(context.Background(), requests.SignupRequest{
		Email:         "existing@example.com",
		CompanyName:   "Company",
		TenantSlug:    "company",
		AdminName:     "Admin",
		AdminPassword: "password123",
	}, "")

	require.NotNil(t, resp)
	assert.Equal(t, responses.StatusConflict, resp.StatusCode)
}

func TestSignupService_InitiateSignup_RepositoryError(t *testing.T) {
	repo := &mockSignupRepo{
		initiateResp: &responses.InternalResponse{
			Error:   errors.New("db error"),
			Message: "Error interno",
			Handled: false,
		},
	}
	svc := NewSignupService(repo, nil)

	resp := svc.InitiateSignup(context.Background(), requests.SignupRequest{
		Email:         "user@example.com",
		CompanyName:   "Company",
		TenantSlug:    "company",
		AdminName:     "Admin",
		AdminPassword: "password123",
	}, "")

	require.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.False(t, resp.Handled)
}

// ─── VerifySignup ─────────────────────────────────────────────────────────────

func TestSignupService_VerifySignup_Success(t *testing.T) {
	expected := &responses.SignupVerifiedResponse{
		Token:    "jwt-token-abc",
		TenantID: "tenant-uuid-123",
		Email:    "admin@newco.com",
		Name:     "John Doe",
	}
	repo := &mockSignupRepo{
		verifyResp: expected,
		verifyErr:  nil,
	}
	svc := NewSignupService(repo, nil)

	result, errResp := svc.VerifySignup(context.Background(), "valid-token-hex")

	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, expected.Token, result.Token)
	assert.Equal(t, expected.TenantID, result.TenantID)
	assert.Equal(t, expected.Email, result.Email)
}

func TestSignupService_VerifySignup_InvalidToken(t *testing.T) {
	repo := &mockSignupRepo{
		verifyResp: nil,
		verifyErr: &responses.InternalResponse{
			Message:    "El enlace de verificación es inválido o expiró.",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	svc := NewSignupService(repo, nil)

	result, errResp := svc.VerifySignup(context.Background(), "invalid-token")

	assert.Nil(t, result)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
	assert.True(t, errResp.Handled)
}

// ─── mock roles repo (S3.5.6 B22 enrichment) ─────────────────────────────────

type mockSignupRolesRepo struct {
	permissions json.RawMessage
	permErr     error
	roleEntry   *ports.RoleEntry
	roleErr     error
}

func (m *mockSignupRolesRepo) GetRolePermissions(_ context.Context, _ string) ([]byte, error) {
	return m.permissions, m.permErr
}

func (m *mockSignupRolesRepo) List(_ context.Context) ([]ports.RoleEntry, error) {
	return nil, nil
}

func (m *mockSignupRolesRepo) GetByID(_ context.Context, _ string) (*ports.RoleEntry, error) {
	return m.roleEntry, m.roleErr
}

func (m *mockSignupRolesRepo) UpdatePermissions(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}

// TestSignupVerify_ResponseIncludesRoleAndPermissions covers the S3.5.6 B22 fix.
//
// Before: VerifySignup returned only {token, tenant_id, email, name}, so the
// frontend's auto-login (ingestExternalToken) left auth_estock.role + permissions
// undefined → menu collapsed to Dashboard → user trapped until logout+login.
//
// After: when RolesRepository is wired, the service populates role (string name)
// and permissions (raw JSON), mirroring AuthenticationService.Login. This test
// pins that contract so future refactors can't silently regress B22.
func TestSignupVerify_ResponseIncludesRoleAndPermissions(t *testing.T) {
	perms := json.RawMessage(`{"all":true}`)
	repo := &mockSignupRepo{
		verifyResp: &responses.SignupVerifiedResponse{
			Token:    "jwt-admin",
			TenantID: "tenant-uuid-xyz",
			Email:    "owner@newco.com",
			Name:     "New Co Admin",
			RoleID:   "role-admin-uuid",
		},
	}
	rolesRepo := &mockSignupRolesRepo{
		permissions: perms,
		roleEntry:   &ports.RoleEntry{ID: "role-admin-uuid", Name: "Admin"},
	}
	svc := NewSignupService(repo, rolesRepo)

	result, errResp := svc.VerifySignup(context.Background(), "verify-token")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "Admin", result.Role, "role name must be resolved from RolesRepository")
	assert.Equal(t, perms, result.Permissions, "permissions JSON must come through verbatim")
	// Sanity: original fields preserved
	assert.Equal(t, "jwt-admin", result.Token)
	assert.Equal(t, "tenant-uuid-xyz", result.TenantID)
}

// When the roles repo is missing (legacy wiring) the service must not panic and
// must return the unenriched payload — frontend recovers via logout+login.
func TestSignupVerify_NoRolesRepo_PassthroughResponse(t *testing.T) {
	repo := &mockSignupRepo{
		verifyResp: &responses.SignupVerifiedResponse{
			Token:  "jwt-admin",
			RoleID: "role-admin-uuid",
		},
	}
	svc := NewSignupService(repo, nil)

	result, errResp := svc.VerifySignup(context.Background(), "verify-token")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "", result.Role)
	assert.Nil(t, result.Permissions)
}

// Permissions lookup error must not break the response — JWT is still valid.
func TestSignupVerify_PermissionsLookupError_StillReturnsToken(t *testing.T) {
	repo := &mockSignupRepo{
		verifyResp: &responses.SignupVerifiedResponse{
			Token:  "jwt-admin",
			RoleID: "role-admin-uuid",
		},
	}
	rolesRepo := &mockSignupRolesRepo{
		permErr:   errors.New("db down"),
		roleEntry: &ports.RoleEntry{ID: "role-admin-uuid", Name: "Admin"},
	}
	svc := NewSignupService(repo, rolesRepo)

	result, errResp := svc.VerifySignup(context.Background(), "verify-token")
	require.Nil(t, errResp)
	require.NotNil(t, result)
	assert.Equal(t, "jwt-admin", result.Token, "JWT must survive even if permissions fetch fails")
	assert.Equal(t, "Admin", result.Role, "role name still resolved despite permissions error")
	assert.Nil(t, result.Permissions, "permissions skipped on lookup error")
}

func TestSignupService_VerifySignup_TransactionError(t *testing.T) {
	repo := &mockSignupRepo{
		verifyResp: nil,
		verifyErr: &responses.InternalResponse{
			Error:   errors.New("tx error"),
			Message: "Error al completar registro",
			Handled: false,
		},
	}
	svc := NewSignupService(repo, nil)

	result, errResp := svc.VerifySignup(context.Background(), "valid-token")

	assert.Nil(t, result)
	require.NotNil(t, errResp)
	assert.NotNil(t, errResp.Error)
	assert.False(t, errResp.Handled)
}
