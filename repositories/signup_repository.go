package repositories

import (
	"context"
	"errors"
	"fmt"
	"html"
	"regexp"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

var slugRegexp = regexp.MustCompile(`^[a-z0-9-]{3,32}$`)

// SignupRepository implements ports.SignupRepository using GORM.
type SignupRepository struct {
	DB          *gorm.DB
	Config      configuration.Config
	EmailSender tools.EmailSender
}

// InitiateSignup validates uniqueness, creates a pending signup token, and sends a verification email.
//
// TODO(M7 — S3.5): expired signup_tokens (expires_at < NOW(), used_at IS NULL) accumulate indefinitely.
// Add a CronDispatch cleanup job:
//   DELETE FROM signup_tokens WHERE expires_at < NOW() - INTERVAL '7 days'
// Deferred to S3.5; at current signup volume the table won't grow to problematic size in the near term.
func (r *SignupRepository) InitiateSignup(ctx context.Context, req requests.SignupRequest) *responses.InternalResponse {
	// Extra validation: slug pattern (validator tag handles min/max length but not regex).
	if !slugRegexp.MatchString(req.TenantSlug) {
		return &responses.InternalResponse{
			Message:    "tenant_slug debe contener solo letras minúsculas, números y guiones (3-32 caracteres)",
			Handled:    true,
			StatusCode: responses.StatusBadRequest,
		}
	}

	// Check email uniqueness in users table.
	var emailCount int64
	if err := r.DB.WithContext(ctx).Model(&database.User{}).
		Where("LOWER(email) = LOWER(?)", req.Email).
		Count(&emailCount).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al verificar email", Handled: false}
	}
	if emailCount > 0 {
		return &responses.InternalResponse{
			Message:    "Ya existe una cuenta registrada con ese email",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		}
	}

	// Check slug uniqueness in tenants table.
	var slugCount int64
	if err := r.DB.WithContext(ctx).Model(&database.Tenant{}).
		Where("slug = ?", req.TenantSlug).
		Count(&slugCount).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al verificar slug", Handled: false}
	}
	if slugCount > 0 {
		return &responses.InternalResponse{
			Message:    "El subdominio ya está en uso, elige otro",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		}
	}

	// Check for an active pending token with same email or slug to avoid spam.
	var pendingCount int64
	if err := r.DB.WithContext(ctx).Model(&database.SignupToken{}).
		Where("(LOWER(email) = LOWER(?) OR tenant_slug = ?) AND used_at IS NULL AND expires_at > NOW()", req.Email, req.TenantSlug).
		Count(&pendingCount).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al verificar solicitud pendiente", Handled: false}
	}
	if pendingCount > 0 {
		return &responses.InternalResponse{
			Message:    "Ya existe una solicitud de registro pendiente para ese email o subdominio. Revisa tu bandeja de entrada.",
			Handled:    true,
			StatusCode: responses.StatusConflict,
		}
	}

	// Generate crypto-random 32-byte hex token.
	token, err := tools.GenerateSecureToken(32)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al generar token", Handled: false}
	}

	// Encrypt the admin password before storing in the pending token (safe at rest).
	encryptedPwd, err := tools.Encrypt(req.AdminPassword, r.Config.JWTSecret)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al procesar contraseña", Handled: false}
	}

	id := uuid.NewString()
	st := database.SignupToken{
		ID:               id,
		Email:            req.Email,
		TenantName:       req.CompanyName,
		TenantSlug:       req.TenantSlug,
		Token:            token,
		AdminName:        req.AdminName,
		AdminPasswordEnc: encryptedPwd,
		ExpiresAt:        time.Now().Add(24 * time.Hour),
	}

	if err := r.DB.WithContext(ctx).Create(&st).Error; err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al crear solicitud de registro", Handled: false}
	}

	// Send verification email — non-blocking on failure (token already persisted).
	appURL := r.Config.AppURL
	if appURL == "" {
		appURL = "http://localhost:4200"
	}
	verifyLink := fmt.Sprintf("%s/verify-signup?token=%s", appURL, token)

	go func() {
		if r.EmailSender == nil {
			return
		}
		subject := "Verifica tu cuenta de eSTOCK"
		htmlBody, textBody := renderSignupVerifyEmail(req.AdminName, req.CompanyName, verifyLink)
		if err := r.EmailSender.Send(context.Background(), req.Email, subject, htmlBody, textBody); err != nil {
			log.Error().Err(err).Str("email", req.Email).Msg("signup verify email send failed")
		}
	}()

	return nil
}

// VerifySignup atomically creates tenant + admin user + demo seed record, then returns a JWT.
func (r *SignupRepository) VerifySignup(ctx context.Context, token string) (*responses.SignupVerifiedResponse, *responses.InternalResponse) {
	// Load the signup token.
	var st database.SignupToken
	err := r.DB.WithContext(ctx).
		Where("token = ? AND used_at IS NULL AND expires_at > NOW()", token).
		First(&st).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &responses.InternalResponse{
				Message:    "El enlace de verificación es inválido o expiró. Solicita un nuevo registro.",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		return nil, &responses.InternalResponse{Error: err, Message: "Error al verificar token", Handled: false}
	}

	var (
		tenantID string
		adminID  string
		adminJWT string
	)

	txErr := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 1. Create tenant.
		tenantID = uuid.NewString()
		tenant := database.Tenant{
			ID:             tenantID,
			Name:           st.TenantName,
			Slug:           st.TenantSlug,
			Email:          st.Email,
			Status:         "trial",
			SignupAt:       now,
			TrialStartedAt: now,
			TrialEndsAt:    now.AddDate(0, 0, 14),
			IsActive:       true,
		}
		if err := tx.Create(&tenant).Error; err != nil {
			return fmt.Errorf("create tenant: %w", err)
		}

		// 2. Find the "admin" role (by name — canonical identifier per S2 roles migration).
		var roleID string
		var role database.Role
		if err := tx.Where("name = ?", "admin").First(&role).Error; err == nil {
			roleID = role.ID
		} else {
			// Fallback: first active role.
			if err2 := tx.Where("is_active = true").First(&role).Error; err2 == nil {
				roleID = role.ID
			}
		}

		// 3. Create admin user using the pre-encrypted password stored in the token.
		adminID = uuid.NewString()
		encPwd := st.AdminPasswordEnc // already Argon2+AES encrypted
		adminName := st.AdminName
		if adminName == "" {
			adminName = st.TenantName + " Admin"
		}
		user := database.User{
			ID:       adminID,
			Name:     adminName,
			Email:    st.Email,
			Password: &encPwd,
			RoleID:   roleID,
			IsActive: true,
		}
		if err := tx.Create(&user).Error; err != nil {
			return fmt.Errorf("create admin user: %w", err)
		}

		// CS6 fix: do NOT pre-insert demo_data_seeds here. The placeholder caused SeedFarma's
		// idempotency guard to find the row and exit immediately — leaving the tenant with an
		// empty WMS. SeedFarma manages demo_data_seeds entirely (checks at start, inserts on
		// success). This tx only creates tenant + user + marks token used.

		// 4. Mark signup token as used.
		if err := tx.Exec("UPDATE signup_tokens SET used_at = NOW() WHERE id = ?", st.ID).Error; err != nil {
			return fmt.Errorf("mark token used: %w", err)
		}

		// 6. Generate JWT for immediate login.
		jwtToken, err := tools.GenerateToken(r.Config.JWTSecret, adminID, adminName, st.Email, roleID)
		if err != nil {
			return fmt.Errorf("generate jwt: %w", err)
		}
		adminJWT = jwtToken
		return nil
	})

	if txErr != nil {
		return nil, &responses.InternalResponse{Error: txErr, Message: "Error al completar registro", Handled: false}
	}

	// After tx: trigger farma demo seed in background goroutine.
	// SeedFarma is idempotent (checks demo_data_seeds before inserting).
	go func(tID string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := tools.SeedFarma(bgCtx, r.DB, tID); err != nil {
			log.Error().Err(err).Str("tenant_id", tID).Msg("farma demo seed failed (background)")
		}
	}(tenantID)

	adminName := st.AdminName
	if adminName == "" {
		adminName = st.TenantName + " Admin"
	}
	return &responses.SignupVerifiedResponse{
		Token:    adminJWT,
		TenantID: tenantID,
		Email:    st.Email,
		Name:     adminName,
	}, nil
}

// ─── email template ───────────────────────────────────────────────────────────

func renderSignupVerifyEmail(adminName, companyName, verifyLink string) (htmlBody, textBody string) {
	// C3 fix: escape user-controlled fields before injecting into HTML to prevent XSS.
	// verifyLink is a server-constructed URL (not user input), so no escaping needed there.
	safeAdminName := html.EscapeString(adminName)
	safeCompanyName := html.EscapeString(companyName)

	text := fmt.Sprintf(
		"Hola %s,\n\nGracias por registrar %s en eSTOCK.\n\nVerifica tu cuenta aquí: %s\n\nEl enlace expira en 24 horas.\n\neSTOCK Team",
		adminName, companyName, verifyLink,
	)
	htmlStr := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:-apple-system,'Plus Jakarta Sans',sans-serif;background:#F0F4FA;margin:0;padding:40px 20px;">
  <div style="max-width:520px;margin:0 auto;background:#fff;border-radius:12px;padding:32px;box-shadow:0 4px 12px rgba(32,49,115,0.08);">
    <h1 style="color:#203173;font-family:Montserrat,sans-serif;font-weight:700;margin:0 0 16px;font-size:24px;">Verifica tu cuenta de eSTOCK</h1>
    <p style="color:#475569;line-height:1.6;margin:0 0 24px;">
      Hola %s,<br><br>Gracias por registrar <strong>%s</strong> en eSTOCK.
      Haz clic en el botón para completar tu registro. El enlace expira en <strong>24 horas</strong>.
    </p>
    <a href="%s" style="display:inline-block;background:#203173;color:#e8d833;padding:12px 32px;border-radius:8px;text-decoration:none;font-weight:600;">Verificar cuenta</a>
    <p style="color:#94A3B8;font-size:12px;margin-top:32px;">Si no solicitaste este registro, puedes ignorar este correo.</p>
  </div>
</body></html>`, safeAdminName, safeCompanyName, verifyLink)
	return htmlStr, text
}
