package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasPermission_EmptyPerms(t *testing.T) {
	assert.False(t, HasPermission(nil, "articles", "read"))
	assert.False(t, HasPermission([]byte{}, "articles", "read"))
}

func TestHasPermission_AdminAll(t *testing.T) {
	perms := []byte(`{"all": true}`)
	assert.True(t, HasPermission(perms, "articles", "read"))
	assert.True(t, HasPermission(perms, "users", "delete"))
}

func TestHasPermission_AdminAllFalse(t *testing.T) {
	perms := []byte(`{"all": false}`)
	assert.False(t, HasPermission(perms, "articles", "read"))
}

func TestHasPermission_SpecificGranted(t *testing.T) {
	perms := []byte(`{"articles": {"read": true, "create": true}}`)
	assert.True(t, HasPermission(perms, "articles", "read"))
	assert.True(t, HasPermission(perms, "articles", "create"))
}

func TestHasPermission_SpecificDenied(t *testing.T) {
	perms := []byte(`{"articles": {"read": true}}`)
	assert.False(t, HasPermission(perms, "articles", "delete"))
	assert.False(t, HasPermission(perms, "users", "read"))
}

func TestHasPermission_ActionFalse(t *testing.T) {
	perms := []byte(`{"articles": {"delete": false}}`)
	assert.False(t, HasPermission(perms, "articles", "delete"))
}

func TestHasPermission_InvalidJSON(t *testing.T) {
	assert.False(t, HasPermission([]byte(`not json`), "articles", "read"))
}
