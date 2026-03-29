package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock auth repo ───────────────────────────────────────────────────────────

type mockAuthRepo struct {
	loginResp *responses.LoginResponse
	loginErr  *responses.InternalResponse
}

func (m *mockAuthRepo) Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse) {
	return m.loginResp, m.loginErr
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func newAuthController(repo *mockAuthRepo) *AuthenticationController {
	svc := services.NewAuthenticationService(repo, nil)
	return NewAuthenticationController(*svc)
}

func doAuthRequest(ctrl *AuthenticationController, body interface{}) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest("POST", "/login", buf)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	ctrl.Login(c)
	return w
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestAuthController_Login_Success(t *testing.T) {
	repo := &mockAuthRepo{
		loginResp: &responses.LoginResponse{
			Name:  "Alice",
			Email: "alice@test.com",
			Token: "jwt-token",
			Role:  "admin",
		},
	}
	ctrl := newAuthController(repo)
	w := doAuthRequest(ctrl, requests.Login{Email: "alice@test.com", Password: "pass123"})
	assert.Equal(t, http.StatusOK, w.Code)

	var resp responses.APIResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Result.Success)
}

func TestAuthController_Login_InvalidCredentials(t *testing.T) {
	repo := &mockAuthRepo{
		loginErr: &responses.InternalResponse{
			Message:    "Credenciales incorrectas",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		},
	}
	ctrl := newAuthController(repo)
	w := doAuthRequest(ctrl, requests.Login{Email: "bad@test.com", Password: "wrong"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Login_ValidationError(t *testing.T) {
	ctrl := newAuthController(&mockAuthRepo{})
	// Missing password
	w := doAuthRequest(ctrl, map[string]string{"email": "not-an-email"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Login_EmptyBody(t *testing.T) {
	ctrl := newAuthController(&mockAuthRepo{})
	w := doAuthRequest(ctrl, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
