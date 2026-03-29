package tools

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key"

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(testSecret, "user-1", "john", "john@test.com", "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	// JWT has 3 dot-separated parts
	assert.Equal(t, 3, len(strings.Split(token, ".")))
}

func TestGetUserId(t *testing.T) {
	token, err := GenerateToken(testSecret, "user-42", "jane", "jane@test.com", "user")
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
	token, err := GenerateToken(testSecret, "u1", "alice", "alice@test.com", "viewer")
	require.NoError(t, err)

	name, err := GetUserName(testSecret, "Bearer "+token)
	require.NoError(t, err)
	assert.Equal(t, "alice", name)
}

func TestGetEmail(t *testing.T) {
	token, err := GenerateToken(testSecret, "u1", "bob", "bob@test.com", "viewer")
	require.NoError(t, err)

	email, err := GetEmail(testSecret, "Bearer "+token)
	require.NoError(t, err)
	assert.Equal(t, "bob@test.com", email)
}

func TestGetRole(t *testing.T) {
	token, err := GenerateToken(testSecret, "u1", "carol", "carol@test.com", "manager")
	require.NoError(t, err)

	role, err := GetRole(testSecret, "Bearer "+token)
	require.NoError(t, err)
	assert.Equal(t, "manager", role)
}
