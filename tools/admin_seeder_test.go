package tools

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ─── helper ─────────────────────────────────────────────────────────────────

// TODO(S3/MA3): Paths 3+4 (integration tests) are skipped in CI (-short mode) because they
// require testcontainers. Two improvements deferred to S3:
//  1. Run integration tests in CI with docker-in-docker testcontainers setup, OR add an
//     `-integration` build tag so they can be opted in selectively.
//  2. Path 4 (DevFreshCreate) should assert that the stored password_hash round-trips
//     correctly through tools.ComparePasswords — currently only asserts user.Email exists.
func setupSeederTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := t.Context()
	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	cleanup := func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	migrationPath := filepath.Join(dir, "..", "db", "migrations")
	require.NoError(t, RunMigrations("file://"+filepath.ToSlash(migrationPath), connStr))

	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{})
	require.NoError(t, err)

	return db, cleanup
}

// seederJWTSecret is a 32-char secret used only in seeder tests.
const seederJWTSecret = "seeder-test-secret-32-chars-xxxx"

// ─── Path 1: not dev — resolveAdminCredentials returns empty strings ──────────

// TestResolveAdminCredentials_NotDev verifies that no credentials are returned
// for non-development environments when env vars are not set.
// This maps to D1 path 1: APP_ENV != development → EnsureDefaultAdmin skips seed.
func TestResolveAdminCredentials_NotDev(t *testing.T) {
	t.Setenv("DEFAULT_ADMIN_EMAIL", "")
	t.Setenv("DEFAULT_ADMIN_PASSWORD", "")

	cfg := configuration.Config{
		Environment:          "production",
		DefaultAdminEmail:    "",
		DefaultAdminPassword: "",
		JWTSecret:            seederJWTSecret,
	}

	email, password := resolveAdminCredentials(cfg)
	assert.Empty(t, email, "should return empty email in non-dev mode without env vars")
	assert.Empty(t, password, "should return empty password in non-dev mode without env vars")
}

// ─── Path 2: dev, no explicit env vars — resolveAdminCredentials uses dev defaults ──

// TestResolveAdminCredentials_DevNoEnv verifies that the development fallback
// credentials (dev@local.test / 12345678) are returned when ENVIRONMENT=development
// and DEFAULT_ADMIN_EMAIL / DEFAULT_ADMIN_PASSWORD are not set.
// This maps to D1 path 2: dev + no explicit credentials → use safe dev defaults.
func TestResolveAdminCredentials_DevNoEnv(t *testing.T) {
	cfg := configuration.Config{
		Environment:          "development",
		DefaultAdminEmail:    "",
		DefaultAdminPassword: "",
		JWTSecret:            seederJWTSecret,
	}

	email, password := resolveAdminCredentials(cfg)
	assert.Equal(t, devDefaultAdminEmail, email, "should use dev fallback email")
	assert.Equal(t, devDefaultAdminPassword, password, "should use dev fallback password")
}

// ─── Path 3 (integration): dev, admin already exists → skip ──────────────────

// TestEnsureDefaultAdmin_DevAdminExists verifies that EnsureDefaultAdmin is
// idempotent: if an admin user already exists, no duplicate is created.
// This maps to D1 path 3: dev + credentials set + admin already in DB → skip.
func TestEnsureDefaultAdmin_DevAdminExists(t *testing.T) {
	db, cleanup := setupSeederTestDB(t)
	defer cleanup()

	cfg := configuration.Config{
		Environment:          "development",
		DefaultAdminEmail:    "existingadmin@test.com",
		DefaultAdminPassword: "password123",
		JWTSecret:            seederJWTSecret,
	}

	// First call creates the admin.
	EnsureDefaultAdmin(db, cfg)

	// Count admins before second call.
	adminRoleID := resolveAdminRoleID(db)
	require.NotEmpty(t, adminRoleID, "admin role must exist after migrations")

	var countBefore int64
	require.NoError(t, db.Model(adminUser()).Where("role_id = ? AND deleted_at IS NULL", adminRoleID).Count(&countBefore).Error)

	// Second call — must be a no-op (idempotent).
	EnsureDefaultAdmin(db, cfg)

	var countAfter int64
	require.NoError(t, db.Model(adminUser()).Where("role_id = ? AND deleted_at IS NULL", adminRoleID).Count(&countAfter).Error)

	assert.Equal(t, countBefore, countAfter, "admin count should not increase on second seeder call")
}

// ─── Path 4 (integration): dev, no admin → creates admin ─────────────────────

// TestEnsureDefaultAdmin_DevFreshCreate verifies that EnsureDefaultAdmin creates
// an admin user with the configured credentials when no admin exists.
// This maps to D1 path 4: dev + credentials set + no admin in DB → create.
func TestEnsureDefaultAdmin_DevFreshCreate(t *testing.T) {
	db, cleanup := setupSeederTestDB(t)
	defer cleanup()

	cfg := configuration.Config{
		Environment:          "development",
		DefaultAdminEmail:    "newadmin@estock.dev",
		DefaultAdminPassword: "admin-secret-32chars-placeholder",
		JWTSecret:            seederJWTSecret,
	}

	adminRoleID := resolveAdminRoleID(db)
	require.NotEmpty(t, adminRoleID, "admin role must exist after migrations")

	// Confirm no admin exists yet.
	var countBefore int64
	require.NoError(t, db.Model(adminUser()).Where("role_id = ? AND deleted_at IS NULL", adminRoleID).Count(&countBefore).Error)
	require.Equal(t, int64(0), countBefore, "precondition: no admin should exist")

	// Seed.
	EnsureDefaultAdmin(db, cfg)

	// Verify admin was created.
	var countAfter int64
	require.NoError(t, db.Model(adminUser()).Where("role_id = ? AND deleted_at IS NULL", adminRoleID).Count(&countAfter).Error)
	assert.Equal(t, int64(1), countAfter, "EnsureDefaultAdmin should have created exactly one admin")

	// Verify the created user has correct email.
	var email string
	require.NoError(t, db.Table("users").
		Joins("JOIN roles ON users.role_id = roles.id").
		Where("LOWER(roles.name) = 'admin' AND users.deleted_at IS NULL").
		Limit(1).
		Pluck("users.email", &email).Error)
	assert.Equal(t, cfg.DefaultAdminEmail, email)
}

// adminUser returns a zero-value User for use in GORM Model() calls.
// Avoids importing the database package directly in the test helper.
func adminUser() interface{} {
	type User struct {
		ID    string `gorm:"column:id;primaryKey"`
		Email string `gorm:"column:email"`
	}
	return &User{}
}
