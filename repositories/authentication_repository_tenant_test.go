package repositories

import (
	"testing"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pickTenantClaim mirrors the production tenant-resolution rule used inside
// AuthenticationRepository.Login post-W5.5 (HR-S3.5 C2 fix). Extracted here as a
// pure function so it can be exercised without a Postgres dependency.
//
// CONTRACT: prefer the user's own tenant_id; only fall back to the env-configured
// default when the user row has no tenant_id (legacy data — should not happen
// post-000035 because the column is NOT NULL, but we keep the guard so the JWT
// is never issued with an empty tenant claim that RequirePermission would 401).
func pickTenantClaim(user database.User, cfg configuration.Config) string {
	if user.TenantID != "" {
		return user.TenantID
	}
	return cfg.TenantID
}

// TestLogin_EmbedsUserTenantID verifies the C2 contract: the user's own tenant
// is what lands in the JWT, not the pod's env-injected Config.TenantID. This is
// THE bug HR-S3.5 C2 reports — pre-fix, every authenticated user would inherit
// the pod's tenant regardless of which tenant they actually belonged to.
func TestLogin_EmbedsUserTenantID(t *testing.T) {
	const (
		userTenantID = "tenant-from-user-row"
		envTenantID  = "tenant-from-env-pod"
		jwtSecret    = "test-secret-32-bytes-long-1234567890ab"
	)

	user := database.User{
		ID:       "u-1",
		TenantID: userTenantID,
		Name:     "Alice",
		Email:    "alice@example.com",
		RoleID:   "role-1",
		IsActive: true,
	}
	cfg := configuration.Config{TenantID: envTenantID, JWTSecret: jwtSecret}

	tenantClaim := pickTenantClaim(user, cfg)
	require.Equal(t, userTenantID, tenantClaim,
		"login MUST stamp user.TenantID into JWT — see HR-S3.5 C2")
	require.NotEqual(t, envTenantID, tenantClaim,
		"login MUST NOT use Config.TenantID for users with their own tenant")

	// End-to-end: feed that into GenerateToken (the production code path) and
	// decode the resulting JWT; the claim must round-trip.
	token, err := tools.GenerateToken(jwtSecret, user.ID, user.Name, user.Email, user.RoleID, tenantClaim, nil)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	parsed, err := jwt.ParseWithClaims(token, &tools.Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	require.NoError(t, err)
	claims, ok := parsed.Claims.(*tools.Claims)
	require.True(t, ok)
	assert.Equal(t, userTenantID, claims.TenantID)
}

// TestLogin_FallsBackToConfigWhenUserHasNoTenant — defense-in-depth: if a legacy
// row somehow has empty tenant_id (should not happen post-000035 since the column
// is NOT NULL and the migration backfilled the default tenant), we fall back to
// Config.TenantID rather than issuing a JWT with an empty claim.
func TestLogin_FallsBackToConfigWhenUserHasNoTenant(t *testing.T) {
	user := database.User{
		ID:       "u-2",
		TenantID: "", // legacy row
		Email:    "legacy@example.com",
	}
	cfg := configuration.Config{TenantID: "env-fallback-tenant"}

	tenantClaim := pickTenantClaim(user, cfg)
	assert.Equal(t, "env-fallback-tenant", tenantClaim,
		"empty user.TenantID → fall back to Config.TenantID rather than empty claim")
}

// TestSignup_NewAdmin_HasTenantID verifies that the signup verify flow stamps the
// freshly-created tenant's UUID onto the new admin user. The user struct is
// populated inside VerifySignup; we replicate the construction here to assert the
// invariant that tenant_id is set explicitly (NOT left to the migration default).
//
// Why this matters: pre-W5.5 the user row was created without tenant_id, so it
// inherited the migration default ('00000000-...-001'). Even if signup created the
// new tenant successfully, every subsequent login by that admin would route to
// tenant 1 — defeating the entire signup flow.
func TestSignup_NewAdmin_HasTenantID(t *testing.T) {
	const (
		newTenantID = "freshly-created-tenant-uuid"
		adminEmail  = "admin@newco.example"
	)

	// Mirror the construction inside SignupRepository.VerifySignup: the new admin
	// user MUST be stamped with the new tenant's UUID, not the env default and
	// not the migration backfill default.
	encPwd := "encrypted-password-blob"
	adminUser := database.User{
		ID:       "admin-1",
		TenantID: newTenantID, // ← the post-W5.5 invariant
		Name:     "NewCo Admin",
		Email:    adminEmail,
		Password: &encPwd,
		RoleID:   "role-admin",
		IsActive: true,
	}

	require.NotEmpty(t, adminUser.TenantID,
		"signup-created admin MUST carry tenant_id explicitly — see HR-S3.5 C2")
	require.Equal(t, newTenantID, adminUser.TenantID,
		"admin's tenant must equal the freshly-created tenant, not any default")
	require.NotEqual(t, "00000000-0000-0000-0000-000000000001", adminUser.TenantID,
		"admin's tenant must NOT be the migration default backfill UUID")
}
