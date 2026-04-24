package services

import (
	"context"
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── mock signup repo ─────────────────────────────────────────────────────────

type mockSignupRepo struct {
	initiateResp *responses.InternalResponse
	verifyResp   *responses.SignupVerifiedResponse
	verifyErr    *responses.InternalResponse
}

func (m *mockSignupRepo) InitiateSignup(_ context.Context, _ requests.SignupRequest) *responses.InternalResponse {
	return m.initiateResp
}

func (m *mockSignupRepo) VerifySignup(_ context.Context, _ string) (*responses.SignupVerifiedResponse, *responses.InternalResponse) {
	return m.verifyResp, m.verifyErr
}

// ─── InitiateSignup ───────────────────────────────────────────────────────────

func TestSignupService_InitiateSignup_Success(t *testing.T) {
	repo := &mockSignupRepo{initiateResp: nil} // nil = success
	svc := NewSignupService(repo)

	resp := svc.InitiateSignup(context.Background(), requests.SignupRequest{
		Email:         "admin@newco.com",
		CompanyName:   "New Company SA",
		TenantSlug:    "newcompany",
		AdminName:     "John Doe",
		AdminPassword: "supersecret123",
	})

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
	svc := NewSignupService(repo)

	resp := svc.InitiateSignup(context.Background(), requests.SignupRequest{
		Email:         "existing@example.com",
		CompanyName:   "Company",
		TenantSlug:    "company",
		AdminName:     "Admin",
		AdminPassword: "password123",
	})

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
	svc := NewSignupService(repo)

	resp := svc.InitiateSignup(context.Background(), requests.SignupRequest{
		Email:         "user@example.com",
		CompanyName:   "Company",
		TenantSlug:    "company",
		AdminName:     "Admin",
		AdminPassword: "password123",
	})

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
	svc := NewSignupService(repo)

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
	svc := NewSignupService(repo)

	result, errResp := svc.VerifySignup(context.Background(), "invalid-token")

	assert.Nil(t, result)
	require.NotNil(t, errResp)
	assert.Equal(t, responses.StatusBadRequest, errResp.StatusCode)
	assert.True(t, errResp.Handled)
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
	svc := NewSignupService(repo)

	result, errResp := svc.VerifySignup(context.Background(), "valid-token")

	assert.Nil(t, result)
	require.NotNil(t, errResp)
	assert.NotNil(t, errResp.Error)
	assert.False(t, errResp.Handled)
}
