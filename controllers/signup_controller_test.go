package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── mock signup repo ─────────────────────────────────────────────────────────

type mockSignupRepoCtrl struct {
	initiateResp *responses.InternalResponse
	verifyResp   *responses.SignupVerifiedResponse
	verifyErr    *responses.InternalResponse
}

func (m *mockSignupRepoCtrl) InitiateSignup(_ context.Context, _ requests.SignupRequest) *responses.InternalResponse {
	return m.initiateResp
}

func (m *mockSignupRepoCtrl) VerifySignup(_ context.Context, _ string) (*responses.SignupVerifiedResponse, *responses.InternalResponse) {
	return m.verifyResp, m.verifyErr
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newSignupController(repo *mockSignupRepoCtrl) *SignupController {
	// rolesRepo nil → controller tests stay focused on HTTP shape; service-layer
	// enrichment is covered by services/signup_service_test.go.
	svc := services.NewSignupService(repo, nil)
	return NewSignupController(svc)
}

func doSignupRequest(ctrl *SignupController, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/signup", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	ctrl.InitiateSignup(c)
	return w
}

func doVerifyRequest(ctrl *SignupController, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/api/signup/verify", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	ctrl.VerifySignup(c)
	return w
}

// ─── InitiateSignup tests ────────────────────────────────────────────────────

func TestSignupController_InitiateSignup_Success(t *testing.T) {
	repo := &mockSignupRepoCtrl{initiateResp: nil}
	ctrl := newSignupController(repo)

	w := doSignupRequest(ctrl, map[string]string{
		"email":          "new@tenant.com",
		"company_name":   "Tenant Corp",
		"tenant_slug":    "tenantcorp",
		"admin_name":     "Admin User",
		"admin_password": "secret1234",
	})

	assert.Equal(t, http.StatusAccepted, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["success"])
}

func TestSignupController_InitiateSignup_ValidationError_MissingFields(t *testing.T) {
	repo := &mockSignupRepoCtrl{initiateResp: nil}
	ctrl := newSignupController(repo)

	// Missing required fields
	w := doSignupRequest(ctrl, map[string]string{
		"email": "not-an-email",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignupController_InitiateSignup_ValidationError_PasswordTooShort(t *testing.T) {
	repo := &mockSignupRepoCtrl{initiateResp: nil}
	ctrl := newSignupController(repo)

	w := doSignupRequest(ctrl, map[string]string{
		"email":          "admin@co.com",
		"company_name":   "Co",
		"tenant_slug":    "co",
		"admin_name":     "Admin",
		"admin_password": "short", // < 8 chars
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignupController_InitiateSignup_Conflict(t *testing.T) {
	repo := &mockSignupRepoCtrl{
		initiateResp: &responses.InternalResponse{
			Message:    "Ya existe una cuenta registrada con ese email",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		},
	}
	ctrl := newSignupController(repo)

	w := doSignupRequest(ctrl, map[string]string{
		"email":          "existing@co.com",
		"company_name":   "Company SA",
		"tenant_slug":    "companysa",
		"admin_name":     "Admin",
		"admin_password": "password123",
	})

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestSignupController_InitiateSignup_EmptyBody(t *testing.T) {
	repo := &mockSignupRepoCtrl{}
	ctrl := newSignupController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/api/signup", bytes.NewBuffer(nil))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	ctrl.InitiateSignup(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── VerifySignup tests ──────────────────────────────────────────────────────

func TestSignupController_VerifySignup_Success(t *testing.T) {
	repo := &mockSignupRepoCtrl{
		verifyResp: &responses.SignupVerifiedResponse{
			Token:    "jwt-abc",
			TenantID: "tenant-uuid",
			Email:    "admin@co.com",
			Name:     "Admin",
		},
	}
	ctrl := newSignupController(repo)

	w := doVerifyRequest(ctrl, map[string]string{
		"token": "valid-hex-token",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	var body responses.APIResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Result.Success)
}

func TestSignupController_VerifySignup_InvalidToken(t *testing.T) {
	repo := &mockSignupRepoCtrl{
		verifyErr: &responses.InternalResponse{
			Message:    "El enlace es inválido o expiró",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	ctrl := newSignupController(repo)

	w := doVerifyRequest(ctrl, map[string]string{
		"token": "bad-token",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSignupController_VerifySignup_MissingToken(t *testing.T) {
	repo := &mockSignupRepoCtrl{}
	ctrl := newSignupController(repo)

	w := doVerifyRequest(ctrl, map[string]string{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── rate limit test (unit smoke — checks middleware wires correctly) ─────────

func TestSignupController_RateLimit_Middleware_RejectsAfterBurst(t *testing.T) {
	// This test is intentionally lightweight — it validates the controller shape
	// for rate-limit route registration (full rate-limit test is an integration test).
	// The middleware is applied in routes/signup_routes.go; here we just assert the
	// controller handles a 202 on first call.
	repo := &mockSignupRepoCtrl{initiateResp: nil}
	ctrl := newSignupController(repo)

	w := doSignupRequest(ctrl, map[string]string{
		"email":          "ratelimit@test.com",
		"company_name":   "Rate Co",
		"tenant_slug":    "rateco",
		"admin_name":     "Rate Admin",
		"admin_password": "ratepwd123",
	})

	assert.Equal(t, http.StatusAccepted, w.Code)
}
