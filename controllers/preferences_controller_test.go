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
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── mock preferences repo ────────────────────────────────────────────────────

type mockPrefsRepo struct {
	prefs     *ports.PreferencesEntry
	getErr    error
	createErr error
	updateErr error
}

func (m *mockPrefsRepo) GetUserPreferences(_ context.Context, _ string) (*ports.PreferencesEntry, error) {
	return m.prefs, m.getErr
}
func (m *mockPrefsRepo) GetOrCreateUserPreferences(_ context.Context, _ string) (*ports.PreferencesEntry, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.prefs != nil {
		return m.prefs, nil
	}
	return &ports.PreferencesEntry{Theme: "system", Language: "es"}, nil
}
func (m *mockPrefsRepo) UpdateUserPreferences(_ context.Context, arg ports.UpdatePreferencesParams) (*ports.PreferencesEntry, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return &ports.PreferencesEntry{
		Theme:    arg.Theme,
		Language: arg.Language,
	}, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func newPrefsContext(method, path string, uid string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	if uid != "" {
		c.Set(tools.ContextKeyUserID, uid)
	}
	return c, w
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestPreferencesController_GetPreferences_Success(t *testing.T) {
	repo := &mockPrefsRepo{
		prefs: &ports.PreferencesEntry{Theme: "dark", Language: "es"},
	}
	ctrl := NewPreferencesController(repo)

	c, w := newPrefsContext("GET", "/preferences", "user-1", nil)
	ctrl.GetPreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreferencesController_GetPreferences_NoUID(t *testing.T) {
	ctrl := NewPreferencesController(&mockPrefsRepo{})

	c, w := newPrefsContext("GET", "/preferences", "", nil)
	ctrl.GetPreferences(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPreferencesController_GetPreferences_NilRepo(t *testing.T) {
	ctrl := NewPreferencesController(nil)

	c, w := newPrefsContext("GET", "/preferences", "user-1", nil)
	ctrl.GetPreferences(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPreferencesController_GetPreferences_CreatesDefault(t *testing.T) {
	// prefs == nil triggers GetOrCreate
	repo := &mockPrefsRepo{prefs: nil}
	ctrl := NewPreferencesController(repo)

	c, w := newPrefsContext("GET", "/preferences", "user-1", nil)
	ctrl.GetPreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreferencesController_UpdatePreferences_Success(t *testing.T) {
	repo := &mockPrefsRepo{prefs: &ports.PreferencesEntry{Theme: "system", Language: "es"}}
	ctrl := NewPreferencesController(repo)

	body := map[string]interface{}{
		"theme":    "dark",
		"language": "es",
	}
	c, w := newPrefsContext("PUT", "/preferences", "user-1", body)
	ctrl.UpdatePreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPreferencesController_UpdatePreferences_NoUID(t *testing.T) {
	ctrl := NewPreferencesController(&mockPrefsRepo{})

	c, w := newPrefsContext("PUT", "/preferences", "", map[string]string{"theme": "dark"})
	ctrl.UpdatePreferences(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPreferencesController_UpdatePreferences_UpdateError(t *testing.T) {
	repo := &mockPrefsRepo{
		prefs:     &ports.PreferencesEntry{},
		updateErr: errors.New("db error"),
	}
	ctrl := NewPreferencesController(repo)

	body := map[string]interface{}{"theme": "dark"}
	c, w := newPrefsContext("PUT", "/preferences", "user-1", body)
	ctrl.UpdatePreferences(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
