package tools

import (
	"strings"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const (
	devDefaultAdminEmail    = "dev@local.test"
	devDefaultAdminPassword = "12345678"
)

// EnsureDefaultAdmin seeds a default Admin user if none exists.
//
// Credential resolution order:
//  1. DEFAULT_ADMIN_EMAIL / DEFAULT_ADMIN_PASSWORD env vars (all environments)
//  2. Hardcoded dev defaults (dev@local.test / 12345678) — only when ENVIRONMENT=development
//  3. Skip silently if neither applies (e.g. production without env vars — ePlatform will provision the user)
//
// Idempotent: does nothing if at least one Admin user already exists.
func EnsureDefaultAdmin(db *gorm.DB, config configuration.Config) {
	email, password := resolveAdminCredentials(config)
	if email == "" || password == "" {
		log.Info().Msg("admin seeder: no credentials configured, skipping")
		return
	}

	// Check if any Admin user already exists
	adminRoleID := resolveAdminRoleID(db)
	if adminRoleID == "" {
		log.Warn().Msg("admin seeder: Admin role not found — migrations may not have run yet")
		return
	}

	var count int64
	if err := db.Model(&database.User{}).Where("role_id = ? AND deleted_at IS NULL", adminRoleID).Count(&count).Error; err != nil {
		log.Error().Err(err).Msg("admin seeder: failed to count admin users")
		return
	}
	if count > 0 {
		log.Info().Int64("existing_admins", count).Msg("admin seeder: admin user already exists, skipping")
		return
	}

	// Encrypt password with JWT secret (AES-GCM — dual-track: dev/qa use reversible encryption
	// so VPS Manager can display credentials. Production users provisioned via ePlatform use argon2id).
	encryptedPassword, err := Encrypt(password, config.JWTSecret)
	if err != nil {
		log.Error().Err(err).Msg("admin seeder: failed to encrypt password")
		return
	}

	name := strings.Split(email, "@")[0]
	now := time.Now()
	// S3.5 W5.5 (HR-S3.5 C2): users.tenant_id is NOT NULL post-000035. Source it from
	// Config.TenantID — the seeded admin belongs to whichever tenant the pod was started
	// for (single-tenant pilot pattern; multi-tenant signups go through SignupRepository
	// which stamps the freshly-created tenant UUID instead).
	tenantID := config.TenantID
	if tenantID == "" {
		tenantID = "00000000-0000-0000-0000-000000000001"
	}
	user := database.User{
		TenantID:  tenantID,
		Name:      name,
		Email:     email,
		FirstName: name,
		LastName:  "",
		Password:  &encryptedPassword,
		RoleID:    adminRoleID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := db.Table(database.User{}.TableName()).Omit("id").Create(&user).Error; err != nil {
		log.Error().Err(err).Str("email", email).Msg("admin seeder: failed to create default admin")
		return
	}

	log.Info().Str("email", email).Str("environment", config.Environment).Msg("admin seeder: default admin created")
}

// resolveAdminCredentials returns the email and password to use for the default admin,
// based on env vars and environment mode.
func resolveAdminCredentials(config configuration.Config) (email, password string) {
	// Explicit env vars take priority in all environments
	if config.DefaultAdminEmail != "" && config.DefaultAdminPassword != "" {
		return config.DefaultAdminEmail, config.DefaultAdminPassword
	}

	// Dev fallback: only when ENVIRONMENT=development
	env := strings.ToLower(config.Environment)
	if env == "development" || env == "debug" {
		email = config.DefaultAdminEmail
		if email == "" {
			email = devDefaultAdminEmail
		}
		password = config.DefaultAdminPassword
		if password == "" {
			password = devDefaultAdminPassword
		}
		return email, password
	}

	return "", ""
}

// resolveAdminRoleID looks up the id of the "Admin" role from the DB.
func resolveAdminRoleID(db *gorm.DB) string {
	var role database.Role
	if err := db.Where("LOWER(name) = 'admin'").First(&role).Error; err != nil {
		return ""
	}
	return role.ID
}
