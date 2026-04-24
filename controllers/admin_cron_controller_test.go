package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── stub roles repository ────────────────────────────────────────────────────

type stubCronRolesRepo struct {
	perms map[string][]byte
}

func (s *stubCronRolesRepo) GetRolePermissions(_ context.Context, role string) ([]byte, error) {
	return s.perms[role], nil
}
func (s *stubCronRolesRepo) List(_ context.Context) ([]ports.RoleEntry, error)             { return nil, nil }
func (s *stubCronRolesRepo) GetByID(_ context.Context, _ string) (*ports.RoleEntry, error) { return nil, nil }
func (s *stubCronRolesRepo) UpdatePermissions(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}

// ─── helper ──────────────────────────────────────────────────────────────────

// performCronRequest wires up JWTAuthMiddleware + RequirePermission + the handler
// on a real gin router so the middleware chain runs correctly.
// routePath is the gin route pattern (no query string); requestURL is the full URL including query.
func performCronRequest(role string, rolesRepo ports.RolesRepository, handler gin.HandlerFunc, method, routePath, requestURL string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	const testSecret = "test-secret-key"

	token, _ := tools.GenerateToken(testSecret, "user-1", "test", "test@test.com", role, "tenant-test")

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(tools.JWTAuthMiddleware(testSecret))
	r.Use(tools.RequirePermission(rolesRepo, "cron", "trigger"))
	r.Handle(method, routePath, handler)

	req, _ := http.NewRequest(method, requestURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	return w
}

// ─── tests ───────────────────────────────────────────────────────────────────

// TestAdminCronTrigger_Admin_RunsJob verifies admin ({"all":true}) gets 200.
// DB is nil, so jobs will error internally, but CronDispatch swallows errors —
// the controller still responds 200 OK.
func TestAdminCronTrigger_Admin_RunsJob(t *testing.T) {
	rolesRepo := &stubCronRolesRepo{
		perms: map[string][]byte{
			"admin": []byte(`{"all":true}`),
		},
	}
	ctrl := &AdminCronController{DB: nil}

	w := performCronRequest("admin", rolesRepo, ctrl.Trigger, "POST", "/admin/cron/trigger", "/admin/cron/trigger?job=all")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAdminCronTrigger_Operator_Forbidden verifies operator (no cron permission) gets 403.
func TestAdminCronTrigger_Operator_Forbidden(t *testing.T) {
	rolesRepo := &stubCronRolesRepo{
		perms: map[string][]byte{
			"operator": []byte(`{"inventory":{"read":true}}`),
		},
	}
	ctrl := &AdminCronController{DB: nil}

	w := performCronRequest("operator", rolesRepo, ctrl.Trigger, "POST", "/admin/cron/trigger", "/admin/cron/trigger")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestAdminCronTrigger_InvalidJob verifies ?job=foo returns 400.
func TestAdminCronTrigger_InvalidJob(t *testing.T) {
	rolesRepo := &stubCronRolesRepo{
		perms: map[string][]byte{
			"admin": []byte(`{"all":true}`),
		},
	}
	ctrl := &AdminCronController{DB: nil}

	w := performCronRequest("admin", rolesRepo, ctrl.Trigger, "POST", "/admin/cron/trigger", "/admin/cron/trigger?job=foo")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminCronTrigger_TrialExpiration_Admin_Returns500OnNilDB verifies that
// ?job=trial_expiration with a nil DB results in a 500 (the job errors internally).
func TestAdminCronTrigger_TrialExpiration_Admin_Returns500OnNilDB(t *testing.T) {
	rolesRepo := &stubCronRolesRepo{
		perms: map[string][]byte{
			"admin": []byte(`{"all":true}`),
		},
	}
	ctrl := &AdminCronController{DB: nil}

	w := performCronRequest("admin", rolesRepo, ctrl.Trigger, "POST", "/admin/cron/trigger", "/admin/cron/trigger?job=trial_expiration")
	// nil DB → RunTrialExpirationCheck returns "cron: nil db" error → ResponseInternal → 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
