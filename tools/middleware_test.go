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
	token, err := GenerateToken(testSecret, "user-1", "alice", "alice@test.com", "admin", "tenant-1")
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
	token, _ := GenerateToken("wrong-secret", "u1", "user", "u@test.com", "user", "tenant-1")

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
	permsMap map[string][]byte
}

func (m *mockPermStore) GetRolePermissions(_ context.Context, roleID string) ([]byte, error) {
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
