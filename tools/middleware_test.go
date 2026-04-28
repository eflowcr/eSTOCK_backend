package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── JWTAuthMiddleware ────────────────────────────────────────────────────────

func TestJWTAuthMiddleware_ValidToken(t *testing.T) {
	token, err := GenerateToken(testSecret, "user-1", "alice", "alice@test.com", "admin", "tenant-1", nil)
	require.NoError(t, err)

	var capturedUID, capturedRole, capturedTenant string
	handler := gin.HandlerFunc(func(c *gin.Context) {
		capturedUID = c.GetString(ContextKeyUserID)
		capturedRole = c.GetString(ContextKeyRole)
		capturedTenant = c.GetString(ContextKeyTenantID)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.Use(JWTAuthMiddleware(testSecret))
	r.GET("/", handler)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	c.Request = req

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", capturedUID)
	assert.Equal(t, "admin", capturedRole)
	assert.Equal(t, "tenant-1", capturedTenant)
}

func TestJWTAuthMiddleware_MissingHeader(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(JWTAuthMiddleware(testSecret))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_InvalidFormat(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(JWTAuthMiddleware(testSecret))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "invalidformat")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuthMiddleware_WrongSecret(t *testing.T) {
	token, _ := GenerateToken("wrong-secret", "u1", "user", "u@test.com", "user", "tenant-1", nil)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(JWTAuthMiddleware(testSecret))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── RequirePermission ────────────────────────────────────────────────────────

type mockPermStore struct {
	permsMap  map[string][]byte
	callCount int // S3.8 — track how often the DB lookup is hit so JWT-cache tests can assert 0 hits
	failErr   error
}

func (m *mockPermStore) GetRolePermissions(_ context.Context, roleID string) ([]byte, error) {
	m.callCount++
	if m.failErr != nil {
		return nil, m.failErr
	}
	if m.permsMap != nil {
		return m.permsMap[roleID], nil
	}
	return nil, nil
}
func (m *mockPermStore) List(_ context.Context) ([]ports.RoleEntry, error)             { return nil, nil }
func (m *mockPermStore) GetByID(_ context.Context, _ string) (*ports.RoleEntry, error) { return nil, nil }
func (m *mockPermStore) UpdatePermissions(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}

func TestRequirePermission_Allowed(t *testing.T) {
	store := &mockPermStore{
		permsMap: map[string][]byte{
			"admin": []byte(`{"articles":{"read":true}}`),
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "admin")
		c.Set(ContextKeyTenantID, "tenant-1")
		c.Next()
	})
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_Forbidden(t *testing.T) {
	store := &mockPermStore{
		permsMap: map[string][]byte{
			"viewer": []byte(`{"articles":{"read":true}}`),
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "viewer")
		c.Set(ContextKeyTenantID, "tenant-1")
		c.Next()
	})
	r.Use(RequirePermission(store, "articles", "delete"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_NoRole(t *testing.T) {
	store := &mockPermStore{}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePermission_NilStore(t *testing.T) {
	// nil store passes through
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(RequirePermission(nil, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_AdminAll(t *testing.T) {
	store := &mockPermStore{
		permsMap: map[string][]byte{
			"superadmin": []byte(`{"all":true}`),
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "superadmin")
		c.Set(ContextKeyTenantID, "tenant-1")
		c.Next()
	})
	r.Use(RequirePermission(store, "anything", "delete"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// S3.5 W3 — pre-W3 tokens (without tenant_id claim) must be rejected so users re-login.
func TestRequirePermission_RejectsTokenWithoutTenantID(t *testing.T) {
	store := &mockPermStore{
		permsMap: map[string][]byte{
			"admin": []byte(`{"articles":{"read":true}}`),
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		// Simulate a pre-W3 token: role is set, tenant_id is NOT.
		c.Set(ContextKeyRole, "admin")
		c.Next()
	})
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "tenant")
}

// Empty-string tenant_id should be treated identically to a missing claim.
func TestRequirePermission_RejectsEmptyTenantID(t *testing.T) {
	store := &mockPermStore{
		permsMap: map[string][]byte{
			"admin": []byte(`{"articles":{"read":true}}`),
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "admin")
		c.Set(ContextKeyTenantID, "")
		c.Next()
	})
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── S3.8 — JWT-cache fast path / DB fallback ────────────────────────────────

// When the JWT carries a permissions claim, RequirePermission MUST authorize off the
// claim and skip the DB lookup entirely. We make the store a tripwire: any DB hit fails
// the test (callCount must stay at 0).
func TestRequirePermission_FromJWTContext_NoDBHit(t *testing.T) {
	store := &mockPermStore{
		// Intentionally wrong perms in DB — if middleware falls back to DB the request
		// would be denied; if it correctly reads JWT it should succeed.
		permsMap: map[string][]byte{
			"admin": []byte(`{}`), // no articles permission in DB
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "admin")
		c.Set(ContextKeyTenantID, "tenant-1")
		// JWT-embedded permissions blob — grants the access.
		c.Set(ContextKeyPermissions, json.RawMessage(`{"articles":{"read":true}}`))
		c.Next()
	})
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, store.callCount, "DB should NOT be hit when JWT carries permissions claim")
	assert.Equal(t, "jwt", w.Header().Get("X-Perm-Source"))
}

// JWT-cache path also returns 403 when the embedded blob doesn't grant the action,
// and importantly, still doesn't hit the DB (middleware never recovers via DB on 403).
func TestRequirePermission_FromJWTContext_DenialDoesNotFallbackToDB(t *testing.T) {
	store := &mockPermStore{
		// DB says yes — but the JWT says no. JWT wins (denial), DB is never consulted.
		permsMap: map[string][]byte{
			"viewer": []byte(`{"articles":{"delete":true}}`),
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "viewer")
		c.Set(ContextKeyTenantID, "tenant-1")
		c.Set(ContextKeyPermissions, json.RawMessage(`{"articles":{"read":true}}`))
		c.Next()
	})
	r.Use(RequirePermission(store, "articles", "delete"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, 0, store.callCount, "DB MUST NOT be consulted to recover from JWT denial — that would leak stale grants")
}

// Backwards compat: a token issued before S3.8 carries no permissions claim, so
// PermissionsFromContext returns nil and the middleware MUST fall back to the DB lookup.
func TestRequirePermission_FallbackToDBLookup_WhenJWTLacksPermissions(t *testing.T) {
	store := &mockPermStore{
		permsMap: map[string][]byte{
			"admin": []byte(`{"articles":{"read":true}}`),
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "admin")
		c.Set(ContextKeyTenantID, "tenant-1")
		// Intentionally do NOT set ContextKeyPermissions — simulates pre-S3.8 token.
		c.Next()
	})
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, store.callCount, "DB lookup MUST run when JWT lacks the permissions claim (legacy/pre-S3.8 token)")
	assert.Equal(t, "db", w.Header().Get("X-Perm-Source"))
}

// An empty permissions blob in the context is treated as "no claim" → DB fallback.
// PermissionsFromContext normalizes len==0 to nil, so this exercises that contract.
func TestRequirePermission_EmptyPermissionsClaim_FallsBackToDB(t *testing.T) {
	store := &mockPermStore{
		permsMap: map[string][]byte{
			"admin": []byte(`{"articles":{"read":true}}`),
		},
	}
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "admin")
		c.Set(ContextKeyTenantID, "tenant-1")
		c.Set(ContextKeyPermissions, json.RawMessage(``)) // empty blob
		c.Next()
	})
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, store.callCount)
}

// End-to-end: token issued via GenerateToken with permissions param round-trips through
// JWTAuthMiddleware → context, and RequirePermission authorizes off the JWT claim.
func TestEndToEnd_TokenWithPermissions_BypassesDB(t *testing.T) {
	perms := json.RawMessage(`{"articles":{"read":true}}`)
	token, err := GenerateToken(testSecret, "u1", "alice", "alice@test.com", "admin", "tenant-1", perms)
	require.NoError(t, err)

	store := &mockPermStore{
		permsMap: map[string][]byte{"admin": []byte(`{}`)}, // tripwire — must not be consulted
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(JWTAuthMiddleware(testSecret))
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, store.callCount)
	assert.Equal(t, "jwt", w.Header().Get("X-Perm-Source"))
}

// End-to-end: legacy token (no permissions param) → JWT carries no claim → DB fallback runs.
func TestEndToEnd_LegacyToken_UsesDBFallback(t *testing.T) {
	token, err := GenerateToken(testSecret, "u1", "alice", "alice@test.com", "admin", "tenant-1", nil)
	require.NoError(t, err)

	store := &mockPermStore{
		permsMap: map[string][]byte{"admin": []byte(`{"articles":{"read":true}}`)},
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(JWTAuthMiddleware(testSecret))
	r.Use(RequirePermission(store, "articles", "read"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, store.callCount, "legacy token MUST trigger DB fallback")
	assert.Equal(t, "db", w.Header().Get("X-Perm-Source"))
}

// PermissionsFromContext contract tests (mirror TenantIDFromContext suite).
func TestPermissionsFromContext_ReturnsClaimValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	blob := json.RawMessage(`{"all":true}`)
	c.Set(ContextKeyPermissions, blob)
	got := PermissionsFromContext(c)
	assert.Equal(t, blob, got)
}

func TestPermissionsFromContext_NilWhenAbsent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assert.Nil(t, PermissionsFromContext(c))
}

func TestPermissionsFromContext_NilWhenWrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ContextKeyPermissions, "not-a-rawmessage")
	assert.Nil(t, PermissionsFromContext(c))
}

func TestPermissionsFromContext_NilWhenEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ContextKeyPermissions, json.RawMessage(``))
	assert.Nil(t, PermissionsFromContext(c))
}
