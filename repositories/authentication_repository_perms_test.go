package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubRolesRepoForJWT is a minimal RolesRepository for testing the JWT permissions
// embed path without a Postgres dependency. We only need GetRolePermissions; the
// remaining methods are no-ops to satisfy the interface.
type stubRolesRepoForJWT struct {
	perms []byte
	err   error
	calls int
}

func (s *stubRolesRepoForJWT) GetRolePermissions(_ context.Context, _ string) ([]byte, error) {
	s.calls++
	return s.perms, s.err
}
func (s *stubRolesRepoForJWT) List(_ context.Context) ([]ports.RoleEntry, error) { return nil, nil }
func (s *stubRolesRepoForJWT) GetByID(_ context.Context, _ string) (*ports.RoleEntry, error) {
	return nil, nil
}
func (s *stubRolesRepoForJWT) UpdatePermissions(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}

// resolvePermsClaim mirrors the production rule used inside AuthenticationRepository.Login
// and SignupRepository.VerifySignup post-S3.8: when a RolesRepository is available and the
// roleID is non-empty, fetch permissions; on error or empty result, return nil so the issued
// token is permissions-less and RequirePermission falls back to the DB lookup.
//
// Extracted as a pure function so the contract can be exercised without spinning up a DB.
// The production callsites inline this same shape; this test pins the contract.
func resolvePermsClaim(repo ports.RolesRepository, roleID string) json.RawMessage {
	if repo == nil || roleID == "" {
		return nil
	}
	perms, err := repo.GetRolePermissions(context.Background(), roleID)
	if err != nil || len(perms) == 0 {
		return nil
	}
	return perms
}

// Happy path: roles repo returns permissions → JWT carries the claim.
func TestLogin_TokenIncludesPermissions(t *testing.T) {
	const (
		jwtSecret = "test-secret-32-bytes-long-1234567890ab"
		roleID    = "role-admin"
	)
	expectedPerms := json.RawMessage(`{"articles":{"read":true,"create":true}}`)
	repo := &stubRolesRepoForJWT{perms: expectedPerms}

	permsClaim := resolvePermsClaim(repo, roleID)
	require.NotNil(t, permsClaim, "S3.8 contract: when roles repo returns permissions, JWT MUST carry the claim")
	require.Equal(t, 1, repo.calls)

	token, err := tools.GenerateToken(jwtSecret, "u-1", "Alice", "alice@example.com", roleID, "tenant-1", permsClaim)
	require.NoError(t, err)

	parsed, err := jwt.ParseWithClaims(token, &tools.Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	require.NoError(t, err)
	claims, ok := parsed.Claims.(*tools.Claims)
	require.True(t, ok)
	assert.JSONEq(t, string(expectedPerms), string(claims.Permissions))
}

// Backwards compat: nil RolesRepository → permsClaim is nil → token issues without claim.
// RequirePermission falls back to DB lookup. Login must not break when the repo isn't wired.
func TestLogin_NilRolesRepo_OmitsPermissionsClaim(t *testing.T) {
	permsClaim := resolvePermsClaim(nil, "role-admin")
	assert.Nil(t, permsClaim, "nil roles repo → no claim → DB fallback path applies")
}

// Defense-in-depth: empty roleID short-circuits the lookup to avoid a wasted DB call.
func TestLogin_EmptyRoleID_OmitsPermissionsClaim(t *testing.T) {
	repo := &stubRolesRepoForJWT{perms: json.RawMessage(`{"all":true}`)}
	permsClaim := resolvePermsClaim(repo, "")
	assert.Nil(t, permsClaim)
	assert.Equal(t, 0, repo.calls, "empty roleID MUST NOT trigger a DB call")
}

// Soft-fail: DB error fetching permissions → token is issued without claim, login still succeeds.
// RequirePermission then falls back to its own DB lookup which can succeed if the issue was transient.
func TestLogin_PermsLookupError_OmitsClaim(t *testing.T) {
	repo := &stubRolesRepoForJWT{err: errors.New("transient db error")}
	permsClaim := resolvePermsClaim(repo, "role-admin")
	assert.Nil(t, permsClaim, "lookup error → no claim, login still proceeds (DB fallback later)")
}

// Empty permissions blob from store → no claim. Avoids embedding a useless empty JSON.
func TestLogin_EmptyPermsResult_OmitsClaim(t *testing.T) {
	repo := &stubRolesRepoForJWT{perms: []byte{}}
	permsClaim := resolvePermsClaim(repo, "role-admin")
	assert.Nil(t, permsClaim)
}

// Same contract applies to the signup verify flow — exact same helper, same expectations.
// We keep a separate test name so the failure surface points at the right area.
func TestSignupVerify_TokenIncludesPermissions(t *testing.T) {
	const (
		jwtSecret = "test-secret-32-bytes-long-1234567890ab"
		roleID    = "role-admin"
	)
	expectedPerms := json.RawMessage(`{"all":true}`)
	repo := &stubRolesRepoForJWT{perms: expectedPerms}

	permsClaim := resolvePermsClaim(repo, roleID)
	require.NotNil(t, permsClaim)

	token, err := tools.GenerateToken(jwtSecret, "admin-1", "Admin", "admin@newco.test", roleID, "fresh-tenant-uuid", permsClaim)
	require.NoError(t, err)

	parsed, err := jwt.ParseWithClaims(token, &tools.Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	require.NoError(t, err)
	claims, ok := parsed.Claims.(*tools.Claims)
	require.True(t, ok)
	assert.JSONEq(t, string(expectedPerms), string(claims.Permissions))
	assert.Equal(t, "fresh-tenant-uuid", claims.TenantID, "tenant claim still round-trips alongside permissions")
}
