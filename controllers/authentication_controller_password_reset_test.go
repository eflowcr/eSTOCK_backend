package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func doForgotPasswordRequest(ctrl *AuthenticationController, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest("POST", "/forgot-password", buf)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	ctrl.ForgotPassword(c)
	return w
}

func doResetPasswordRequest(ctrl *AuthenticationController, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest("POST", "/reset-password", buf)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	ctrl.ResetPassword(c)
	return w
}

// ─── ForgotPassword tests ─────────────────────────────────────────────────────

// 1. Valid email — always returns 200 OK (generic response, regardless of whether user exists).
func TestForgotPassword_EmailExists(t *testing.T) {
	repo := &mockAuthRepo{requestResetResp: nil} // nil = success
	ctrl := newAuthController(repo)
	w := doForgotPasswordRequest(ctrl, requests.ForgotPasswordRequest{Email: "user@eprac.com"})
	assert.Equal(t, http.StatusOK, w.Code)

	var resp responses.APIResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Result.Success)
}

// 2. Unknown email — still 200 OK (prevents user enumeration).
func TestForgotPassword_EmailNotExists(t *testing.T) {
	repo := &mockAuthRepo{requestResetResp: nil}
	ctrl := newAuthController(repo)
	w := doForgotPasswordRequest(ctrl, requests.ForgotPasswordRequest{Email: "nobody@nowhere.com"})
	assert.Equal(t, http.StatusOK, w.Code)
}

// 3. Inactive user — still 200 OK (generic response).
func TestForgotPassword_EmailInactive(t *testing.T) {
	repo := &mockAuthRepo{requestResetResp: nil}
	ctrl := newAuthController(repo)
	w := doForgotPasswordRequest(ctrl, requests.ForgotPasswordRequest{Email: "inactive@eprac.com"})
	assert.Equal(t, http.StatusOK, w.Code)
}

// 4. Invalid email format — 400 Bad Request from validation.
func TestForgotPassword_InvalidEmail(t *testing.T) {
	ctrl := newAuthController(&mockAuthRepo{})
	w := doForgotPasswordRequest(ctrl, map[string]string{"email": "not-an-email"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 5. Missing body — 400 Bad Request.
func TestForgotPassword_EmptyBody(t *testing.T) {
	ctrl := newAuthController(&mockAuthRepo{})
	w := doForgotPasswordRequest(ctrl, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ResetPassword tests ──────────────────────────────────────────────────────

// 6. Valid token — 200 OK.
func TestResetPassword_ValidToken(t *testing.T) {
	repo := &mockAuthRepo{resetPasswordResp: nil}
	ctrl := newAuthController(repo)
	w := doResetPasswordRequest(ctrl, requests.ResetPasswordRequest{
		Token:       "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", // 64-char hex
		NewPassword: "NewSecurePass123",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	var resp responses.APIResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Result.Success)
}

// 7. Expired or invalid token — repo returns 400 Bad Request.
func TestResetPassword_ExpiredToken(t *testing.T) {
	repo := &mockAuthRepo{
		resetPasswordResp: &responses.InternalResponse{
			Message:    "El enlace es inválido o expiró. Solicita uno nuevo.",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	ctrl := newAuthController(repo)
	w := doResetPasswordRequest(ctrl, requests.ResetPasswordRequest{
		Token:       "expiredtokenexpiredtokenexpiredtokenexpiredtokenexpiredtokenexpired",
		NewPassword: "NewSecurePass123",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 8. Already-used token — 400 Bad Request.
func TestResetPassword_UsedToken(t *testing.T) {
	repo := &mockAuthRepo{
		resetPasswordResp: &responses.InternalResponse{
			Message:    "El enlace es inválido o expiró. Solicita uno nuevo.",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	ctrl := newAuthController(repo)
	w := doResetPasswordRequest(ctrl, requests.ResetPasswordRequest{
		Token:       "usedtokenusedtokenusedtokenusedtokenusedtokenusedtokenusedtokenused",
		NewPassword: "NewSecurePass123",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 9. Random invalid token — 400 Bad Request.
func TestResetPassword_InvalidToken(t *testing.T) {
	repo := &mockAuthRepo{
		resetPasswordResp: &responses.InternalResponse{
			Message:    "El enlace es inválido o expiró. Solicita uno nuevo.",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	ctrl := newAuthController(repo)
	w := doResetPasswordRequest(ctrl, requests.ResetPasswordRequest{
		Token:       "invalidtokeninvalidtokeninvalidtokeninvalidtokeninvalidtokeninvalid",
		NewPassword: "ValidPass123",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// 10. Password too short — 400 validation error (< 8 chars).
func TestResetPassword_ShortPassword(t *testing.T) {
	ctrl := newAuthController(&mockAuthRepo{})
	w := doResetPasswordRequest(ctrl, map[string]interface{}{
		"token":        "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		"new_password": "short",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
