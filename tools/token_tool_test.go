package tools

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key"
const testTenantID = "tenant-test-1"

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(testSecret, "user-1", "john", "john@test.com", "admin", testTenantID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	// JWT has 3 dot-separated parts
	assert.Equal(t, 3, len(strings.Split(token, ".")))
}

func TestGetUserId(t *testing.T) {
	token, err := GenerateToken(testSecret, "user-42", "jane", "jane@test.com", "user", testTenantID)
	require.NoError(t, err)

	t.Run("from raw token", func(t *testing.T) {
		id, err := GetUserId(testSecret, token)
		require.NoError(t, err)
		assert.Equal(t, "user-42", id)
	})

	t.Run("from Bearer token", func(t *testing.T) {
		id, err := GetUserId(testSecret, "Bearer "+token)
		require.NoError(t, err)
		assert.Equal(t, "user-42", id)
	})

	t.Run("wrong secret returns error", func(t *testing.T) {
		_, err := GetUserId("wrong-secret", token)
		assert.Error(t, err)
	})

	t.Run("invalid token returns error", func(t *testing.T) {
		_, err := GetUserId(testSecret, "not.a.token")
		assert.Error(t, err)
	})
}

func TestGetUserName(t *testing.T) {
	token, err := GenerateToken(testSecret, "u1", "alice", "alice@test.com", "viewer", testTenantID)
	require.NoError(t, err)

	name, err := GetUserName(testSecret, "Bearer "+token)
	require.NoError(t, err)
	assert.Equal(t, "alice", name)
}

func TestGetEmail(t *testing.T) {
	token, err := GenerateToken(testSecret, "u1", "bob", "bob@test.com", "viewer", testTenantID)
	require.NoError(t, err)

	email, err := GetEmail(testSecret, "Bearer "+token)
	require.NoError(t, err)
	assert.Equal(t, "bob@test.com", email)
}

func TestGetRole(t *testing.T) {
	token, err := GenerateToken(testSecret, "u1", "carol", "carol@test.com", "manager", testTenantID)
	require.NoError(t, err)

	role, err := GetRole(testSecret, "Bearer "+token)
	require.NoError(t, err)
	assert.Equal(t, "manager", role)
}

// S3.5 W3: tenant_id claim plumbing.

func TestGenerateToken_EmbedsTenantID(t *testing.T) {
	token, err := GenerateToken(testSecret, "u1", "user", "u@test.com", "admin", "tenant-abc")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Validate by routing through the middleware to extract claim onto context.
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	var captured string
	r.Use(JWTAuthMiddleware(testSecret))
	r.GET("/", func(c *gin.Context) {
		captured = TenantIDFromContext(c)
		c.Status(200)
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, "tenant-abc", captured)
}

func TestTenantIDFromContext_ReturnsClaimValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ContextKeyTenantID, "tenant-xyz")
	assert.Equal(t, "tenant-xyz", TenantIDFromContext(c))
}

func TestTenantIDFromContext_EmptyWhenAbsent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assert.Equal(t, "", TenantIDFromContext(c))
}

func TestTenantIDFromContext_EmptyWhenWrongType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ContextKeyTenantID, 12345) // wrong type
	assert.Equal(t, "", TenantIDFromContext(c))
}
