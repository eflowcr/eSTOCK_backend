// Integration tests for S3-W5-A: SaaS self-service signup + demo data seeder.
//
// Requires Docker (testcontainers). Run:
//
//	go test -v ./repositories/... -run TestSignup
//
// The tests exercise the full flow:
//
//	POST /api/signup → signup_token row → email (logged) →
//	POST /api/signup/verify → tenant + user + demo_data_seed row → JWT → login works
package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// testConfig returns a minimal Config suitable for integration tests.
func testConfig() configuration.Config {
	// Must be ≥ 32 chars (validateRequired)
	return configuration.Config{
		JWTSecret:   "integration-test-jwt-secret-32chars!!",
		Environment: "test",
		AppURL:      "http://localhost:4200",
	}
}

// signupRepo returns a SignupRepository wired to db with a logger email sender.
func signupRepo(db *gorm.DB) *SignupRepository {
	return &SignupRepository{
		DB:          db,
		Config:      testConfig(),
		EmailSender: &tools.LoggerEmailSender{},
	}
}

// seedAdminRole inserts an "admin" role so VerifySignup can find it.
func seedAdminRole(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		INSERT INTO roles (id, name, is_active, created_at, updated_at)
		VALUES (?, 'admin', true, NOW(), NOW())
		ON CONFLICT DO NOTHING`, id).Error)
	// Return the actual id (may differ if ON CONFLICT hit)
	var roleID string
	require.NoError(t, db.Raw("SELECT id FROM roles WHERE name = 'admin' LIMIT 1").Scan(&roleID).Error)
	return roleID
}

// ─── Test: Full signup → verify flow ─────────────────────────────────────────

func TestSignup_FullFlow_InitiateAndVerify(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	seedAdminRole(t, db)
	repo := signupRepo(db)
	ctx := context.Background()

	req := requests.SignupRequest{
		Email:         "admin@integrationtest.com",
		CompanyName:   "Integration Corp SA",
		TenantSlug:    "integcorp",
		AdminName:     "John Integration",
		AdminPassword: "TestP@ssw0rd!",
	}

	// Step 1: Initiate signup
	err := repo.InitiateSignup(ctx, req)
	assert.Nil(t, err, "InitiateSignup should succeed")

	// Verify signup_token was created
	var st database.SignupToken
	require.NoError(t, db.Where("LOWER(email) = LOWER(?)", req.Email).First(&st).Error)
	assert.Equal(t, req.TenantSlug, st.TenantSlug)
	assert.NotEmpty(t, st.Token)
	assert.NotEmpty(t, st.AdminPasswordEnc)
	assert.True(t, st.ExpiresAt.After(time.Now()), "token should not be expired")
	assert.Nil(t, st.UsedAt, "token should not be used yet")

	// Step 2: Verify signup
	result, errResp := repo.VerifySignup(ctx, st.Token)
	require.Nil(t, errResp, "VerifySignup should succeed")
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Token, "JWT should be returned")
	assert.NotEmpty(t, result.TenantID)
	assert.Equal(t, req.Email, result.Email)

	// Verify tenant was created
	var tenant database.Tenant
	require.NoError(t, db.Where("id = ?", result.TenantID).First(&tenant).Error)
	assert.Equal(t, req.TenantSlug, tenant.Slug)
	assert.Equal(t, req.CompanyName, tenant.Name)
	assert.Equal(t, "trial", tenant.Status)
	assert.True(t, tenant.IsActive)
	assert.True(t, tenant.TrialEndsAt.After(time.Now()), "trial should not be expired")

	// Verify admin user was created
	var user database.User
	require.NoError(t, db.Where("LOWER(email) = LOWER(?)", req.Email).First(&user).Error)
	assert.True(t, user.IsActive)
	assert.NotNil(t, user.Password)

	// Verify password is valid (can authenticate)
	assert.True(t,
		tools.ComparePasswords(*user.Password, req.AdminPassword, testConfig().JWTSecret),
		"stored encrypted password should match original admin_password",
	)

	// Verify signup token is now marked used
	var usedST database.SignupToken
	require.NoError(t, db.Where("id = ?", st.ID).First(&usedST).Error)
	assert.NotNil(t, usedST.UsedAt, "signup token should be marked as used")

	// Wait briefly for async farma seed goroutine (it runs in background)
	// We just check the demo_data_seeds record was created inside the tx.
	var seedRecord database.DemoDataSeed
	require.NoError(t, db.Where("tenant_id = ? AND seed_name = ?", result.TenantID, tools.FarmaSeedName).
		First(&seedRecord).Error)
	assert.Equal(t, tools.FarmaSeedName, seedRecord.SeedName)
}

// ─── Test: Duplicate email is rejected ───────────────────────────────────────

func TestSignup_InitiateSignup_DuplicateEmail_Rejected(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	seedAdminRole(t, db)
	repo := signupRepo(db)
	ctx := context.Background()

	req := requests.SignupRequest{
		Email:         "duplicate@test.com",
		CompanyName:   "Dup Corp",
		TenantSlug:    "dupcorp",
		AdminName:     "Dup Admin",
		AdminPassword: "duppassword123",
	}

	// First signup
	require.Nil(t, repo.InitiateSignup(ctx, req))

	// Second signup with same email
	req2 := req
	req2.TenantSlug = "dupcorp2"
	errResp := repo.InitiateSignup(ctx, req2)
	require.NotNil(t, errResp)
	assert.Contains(t, errResp.Message, "email")
}

// ─── Test: Duplicate slug is rejected ────────────────────────────────────────

func TestSignup_InitiateSignup_DuplicateSlug_Rejected(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	repo := signupRepo(db)
	ctx := context.Background()

	// Create a tenant with the slug already
	require.NoError(t, db.Exec(`
		INSERT INTO tenants (id, name, slug, email, status, trial_ends_at, is_active)
		VALUES (gen_random_uuid(), 'Existing Tenant', 'existingslug', 'x@x.com', 'trial', NOW()+interval '14 days', true)`).Error)

	req := requests.SignupRequest{
		Email:         "newuser@test.com",
		CompanyName:   "New Company",
		TenantSlug:    "existingslug",
		AdminName:     "Admin",
		AdminPassword: "password123",
	}

	errResp := repo.InitiateSignup(ctx, req)
	require.NotNil(t, errResp)
	assert.Contains(t, errResp.Message, "subdominio")
}

// ─── Test: Invalid token returns 400 ─────────────────────────────────────────

func TestSignup_VerifySignup_InvalidToken_Returns400(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	repo := signupRepo(db)
	ctx := context.Background()

	result, errResp := repo.VerifySignup(ctx, "nonexistent-token-hex")
	assert.Nil(t, result)
	require.NotNil(t, errResp)
	assert.True(t, errResp.Handled)
	assert.Equal(t, 400, errResp.StatusCode)
}

// ─── Test: Expired token returns 400 ─────────────────────────────────────────

func TestSignup_VerifySignup_ExpiredToken_Returns400(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	repo := signupRepo(db)
	ctx := context.Background()

	// Insert an expired token directly.
	id, err := tools.GenerateNanoid(db)
	require.NoError(t, err)
	expiredToken := fmt.Sprintf("expiredtokenvalue%s", id)
	require.NoError(t, db.Exec(`
		INSERT INTO signup_tokens
			(id, email, tenant_name, tenant_slug, token, admin_name, admin_password_enc, expires_at)
		VALUES (?, 'exp@test.com', 'Exp Co', 'expco', ?, 'Admin', 'enc', NOW() - interval '1 hour')`,
		id, expiredToken).Error)

	result, errResp := repo.VerifySignup(ctx, expiredToken)
	assert.Nil(t, result)
	require.NotNil(t, errResp)
	assert.True(t, errResp.Handled)
	assert.Equal(t, 400, errResp.StatusCode)
}

// ─── Test: Slug regex validation ─────────────────────────────────────────────

func TestSignup_InitiateSignup_InvalidSlug_Rejected(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	repo := signupRepo(db)
	ctx := context.Background()

	invalidSlugs := []string{
		"has space",
		"HAS_UPPER",
		"ab",     // too short (< 3)
		"12345678901234567890123456789012x", // too long (> 32)
	}

	for _, slug := range invalidSlugs {
		req := requests.SignupRequest{
			Email:         fmt.Sprintf("user_%s@test.com", "x"),
			CompanyName:   "Test Co",
			TenantSlug:    slug,
			AdminName:     "Admin",
			AdminPassword: "password123",
		}
		errResp := repo.InitiateSignup(ctx, req)
		require.NotNil(t, errResp, "expected error for slug %q", slug)
	}
}

// ─── Test: SeedFarma idempotency ─────────────────────────────────────────────

func TestSeedFarma_Idempotent(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := "00000000-0000-0000-0000-000000000001" // default tenant from migration

	// Run SeedFarma twice — should not error on second run.
	// Empty adminUserID: legacy behaviour (created_by falls back to tenantID); the
	// repository LEFT JOIN keeps rows visible regardless. Idempotency does not
	// depend on the new arg.
	err1 := tools.SeedFarma(ctx, db, tenantID, "")
	require.NoError(t, err1, "first SeedFarma should succeed")

	err2 := tools.SeedFarma(ctx, db, tenantID, "")
	require.NoError(t, err2, "second SeedFarma should be idempotent (no error)")

	// Verify articles count is not doubled. S3.5 W4: SKUs are now tenant-prefixed
	// ("T00000000-RX-001"), so we filter on the prefixed pattern for this tenant
	// instead of bare "RX-%".
	var count int64
	require.NoError(t, db.Model(&database.Article{}).
		Where("tenant_id = ? AND sku LIKE '%RX-%'", tenantID).Count(&count).Error)
	assert.EqualValues(t, 50, count, "exactly 50 farma articles should exist for tenant")
}

// ─── S3.5.2 N1: AssignsAdminRole ─────────────────────────────────────────────
// Regression test for the prod bug where new tenant admins received the
// "Operator" role because the case-sensitive `name = 'admin'` lookup missed
// the canonical capitalized "Admin" row inserted by migration 000016.

func TestSignup_AssignsAdminRole(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	// Do NOT seed an extra lower-case "admin" role — rely on the canonical
	// capitalized "Admin" row from migration 000016. This mirrors prod and
	// reproduces the exact condition that triggered N1.

	// Discover the canonical Admin role id so we can assert against it.
	var adminRole database.Role
	require.NoError(t, db.Where("LOWER(name) = ?", "admin").First(&adminRole).Error,
		"migration 000016 should have inserted Admin role")

	repo := signupRepo(db)
	ctx := context.Background()

	req := requests.SignupRequest{
		Email:         "n1.assignsadmin@test.com",
		CompanyName:   "N1 Admin Co",
		TenantSlug:    "n1adminco",
		AdminName:     "N1 Admin",
		AdminPassword: "TestP@ssw0rd!",
	}
	require.Nil(t, repo.InitiateSignup(ctx, req))

	var st database.SignupToken
	require.NoError(t, db.Where("LOWER(email) = LOWER(?)", req.Email).First(&st).Error)

	result, errResp := repo.VerifySignup(ctx, st.Token)
	require.Nil(t, errResp)
	require.NotNil(t, result)

	var user database.User
	require.NoError(t, db.Where("LOWER(email) = LOWER(?)", req.Email).First(&user).Error)
	assert.Equal(t, adminRole.ID, user.RoleID,
		"new tenant admin must be assigned the Admin role, not Operator/Viewer/etc")
}

// ─── S3.5.2 N1: FailsLoudIfNoAdminRole ───────────────────────────────────────
// Regression test for the silent-fallback bug. With no admin role present the
// signup must fail loudly so ops/devs spot the misconfig instead of inheriting
// a random "first active role".

func TestSignup_FailsLoudIfNoAdminRole(t *testing.T) {
	db, cleanup := setupGORMTestDB(t)
	defer cleanup()

	// Hard-delete all admin-named roles (migration may have inserted one).
	// Cascading FK from users.role_id → roles.id means we also need to detach
	// any users currently pointing at an admin role; the migration's default
	// admin user (if any) is fine to retarget at Operator for the test scope.
	var operatorID string
	require.NoError(t, db.Raw("SELECT id FROM roles WHERE LOWER(name) = 'operator' LIMIT 1").Scan(&operatorID).Error)
	if operatorID != "" {
		require.NoError(t, db.Exec("UPDATE users SET role_id = ? WHERE role_id IN (SELECT id FROM roles WHERE LOWER(name) = 'admin')", operatorID).Error)
	}
	require.NoError(t, db.Exec("DELETE FROM roles WHERE LOWER(name) = 'admin'").Error)

	repo := signupRepo(db)
	ctx := context.Background()

	req := requests.SignupRequest{
		Email:         "n1.failsloud@test.com",
		CompanyName:   "N1 Fails Co",
		TenantSlug:    "n1failsco",
		AdminName:     "N1 Fails",
		AdminPassword: "TestP@ssw0rd!",
	}
	require.Nil(t, repo.InitiateSignup(ctx, req))

	var st database.SignupToken
	require.NoError(t, db.Where("LOWER(email) = LOWER(?)", req.Email).First(&st).Error)

	result, errResp := repo.VerifySignup(ctx, st.Token)
	assert.Nil(t, result, "result must be nil when admin role missing")
	require.NotNil(t, errResp, "VerifySignup must error loud, not silently assign a random role")
	if errResp.Error != nil {
		assert.Contains(t, errResp.Error.Error(), "admin role not found",
			"error should clearly identify the missing-role root cause")
	}

	// Sanity: tenant must NOT have been committed (the tx aborted).
	var tenantCount int64
	require.NoError(t, db.Model(&database.Tenant{}).Where("slug = ?", req.TenantSlug).Count(&tenantCount).Error)
	assert.EqualValues(t, 0, tenantCount, "transaction must have rolled back when admin role missing")
}

// ─── Test: Rate-limit middleware integration (route-level) ───────────────────
// Note: full rate-limit integration uses net/http/httptest with the router.
// This is a unit-level smoke test in signup_controller_test.go.
// The heavy rate-limit test (TCP-level, 6 real requests) is left as a manual
// smoke test to avoid flakiness in CI (timing-sensitive).
