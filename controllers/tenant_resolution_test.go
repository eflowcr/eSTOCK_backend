package controllers

import (
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestResolveTenantID_PrefersJWTClaim verifies the C1 contract: when the JWT
// middleware has placed a tenant_id on the gin.Context, controllers must read
// THAT value, not the env-injected default that lives on the controller struct.
//
// Pre-S3.5 W5.5 every tenant-scoped controller read c.TenantID directly, so a
// pod started for tenant 1 served tenant 1 data to every authenticated request,
// regardless of what tenant the JWT actually belonged to. After this fix, the
// JWT claim wins.
func TestResolveTenantID_PrefersJWTClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(tools.ContextKeyTenantID, "tenant-from-jwt")

	got := tools.ResolveTenantID(c, "tenant-from-env")
	assert.Equal(t, "tenant-from-jwt", got, "JWT claim must override env-injected default")
}

// TestResolveTenantID_FallsBackWhenJWTMissing verifies the cron / system / test
// path: when no JWT context is present, the env fallback is returned. This keeps
// background jobs and pre-W5.5 tests working without a JWT.
func TestResolveTenantID_FallsBackWhenJWTMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	got := tools.ResolveTenantID(c, "tenant-from-env")
	assert.Equal(t, "tenant-from-env", got, "no JWT → fall back to constructor default")
}

// TestResolveTenantID_EmptyWhenBothMissing verifies the safety contract: if there
// is no JWT and no fallback, the helper returns "" and the caller is expected to
// 401 — never silently default to a global tenant.
func TestResolveTenantID_EmptyWhenBothMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	got := tools.ResolveTenantID(c, "")
	assert.Equal(t, "", got, "no JWT and no fallback → empty (caller must 401)")
}

// TestArticlesController_HonorsJWTTenant exercises the full C1 wiring through a
// real controller: the constructor receives one tenant (the env default), the
// request comes in with a different tenant in the JWT, and the controller must
// hand the SERVICE the JWT tenant — not the env one.
func TestArticlesController_HonorsJWTTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := &mockArticlesRepoCtrl{}
	ctrl := newArticlesController(mock) // ctrl.TenantID = testTenantIDCtrl (env default)

	// Simulate a JWT-authenticated request from a DIFFERENT tenant.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(tools.ContextKeyTenantID, "jwt-tenant-99")

	resolved := ctrl.resolveTenantID(c)
	assert.Equal(t, "jwt-tenant-99", resolved,
		"controller must source tenant from JWT, not from c.TenantID — see HR-S3.5 C1")
}

// TestLotsController_HonorsJWTTenant — second spot-check on a different controller
// to confirm the pattern is applied consistently across the C1 surface.
func TestLotsController_HonorsJWTTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := &LotsController{TenantID: "env-default"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(tools.ContextKeyTenantID, "jwt-tenant-42")

	resolved := ctrl.resolveTenantID(c)
	assert.Equal(t, "jwt-tenant-42", resolved)
}

// TestStockAlertsController_FallsBackWithoutJWT — third spot-check covering the
// fallback path (system/cron callers that never see a JWT context).
func TestStockAlertsController_FallsBackWithoutJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := &StockAlertsController{TenantID: "env-default"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// no c.Set for tenant_id

	resolved := ctrl.resolveTenantID(c)
	assert.Equal(t, "env-default", resolved,
		"no JWT → controller falls back to env default (cron/admin path)")
}
