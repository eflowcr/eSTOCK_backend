package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type AuthenticationRepository struct {
	DB           *gorm.DB
	JWTSecret    string
	Config       configuration.Config
	EmailSender  tools.EmailSender
	AuditService *services.AuditService
	// RolesRepository is optional. When set (S3.8+), Login looks up the role's permissions
	// blob and embeds it into the issued JWT so RequirePermission can authorize without
	// a per-request DB roundtrip. When nil, GenerateToken receives nil permissions and the
	// resulting token forces RequirePermission to fall back to the DB lookup (legacy path).
	RolesRepository ports.RolesRepository
}

func (a *AuthenticationRepository) Login(login requests.Login) (*responses.LoginResponse, *responses.InternalResponse) {
	var user database.User

	err := a.DB.Where("email = ?", login.Email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &responses.InternalResponse{
			Error:   errors.New("usuario no encontrado"),
			Message: "Credenciales inválidas",
			Handled: true,
		}
	}

	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al obtener el usuario",
			Handled: false,
		}
	}

	if !user.IsActive {
		return nil, &responses.InternalResponse{
			Error:   errors.New("cuenta inactiva"),
			Message: "Su cuenta está inactiva. Por favor, contacte al administrador.",
			Handled: true,
		}
	}

	if user.Password == nil || !tools.ComparePasswords(*user.Password, login.Password, a.JWTSecret) {
		return nil, &responses.InternalResponse{
			Error:   errors.New("contraseña inválida"),
			Message: "Credenciales inválidas",
			Handled: true,
		}
	}

	// S3.5 W5.5 (HR-S3.5 C2 fix) — embed the user's own tenant_id into the JWT. Pre-W5.5
	// this used Config.TenantID (the env-injected pod default), which silently moved every
	// authenticated user into whichever tenant the pod was started for — defeating the
	// W3 multi-tenant claim plumbing. After 000035_users_tenant_id, every users row carries
	// tenant_id (backfilled to the default tenant for legacy rows; freshly stamped on signup),
	// so the JWT now correctly scopes to the user's own tenant.
	//
	// Defense in depth: if user.TenantID is empty (should never happen post-migration —
	// the column is NOT NULL), we fall back to Config.TenantID rather than issue a token
	// with no tenant claim, since RequirePermission would 401 it anyway.
	tenantClaim := user.TenantID
	if tenantClaim == "" {
		tenantClaim = a.Config.TenantID
	}

	// S3.8 — embed the role's permissions blob into the JWT so RequirePermission
	// can authorize without a per-request DB lookup. Failure here is non-fatal:
	// we issue a permissions-less token and RequirePermission falls back to the
	// DB lookup (legacy path). Avoids breaking login if the roles table is
	// briefly unreachable; permissions remain enforced either way.
	var permsClaim json.RawMessage
	if a.RolesRepository != nil && user.RoleID != "" {
		if perms, permErr := a.RolesRepository.GetRolePermissions(context.Background(), user.RoleID); permErr == nil && len(perms) > 0 {
			permsClaim = perms
		} else if permErr != nil {
			log.Warn().Err(permErr).Str("role_id", user.RoleID).Msg("login: failed to load role permissions for JWT — issuing token without permissions claim (DB fallback will apply)")
		}
	}

	token, err := tools.GenerateToken(a.JWTSecret, user.ID, user.Name, user.Email, user.RoleID, tenantClaim, permsClaim)
	if err != nil {
		return nil, &responses.InternalResponse{
			Error:   err,
			Message: "Error al generar el token",
			Handled: false,
		}
	}

	return &responses.LoginResponse{
		Name:     user.FirstName,
		LastName: user.LastName,
		Email:    user.Email,
		Token:    token,
		Role:     user.RoleID,
	}, nil
}

func (r *AuthenticationRepository) RequestPasswordReset(ctx context.Context, email string) *responses.InternalResponse {
	// 1. Buscar usuario activo por email (case-insensitive)
	var user database.User
	err := r.DB.Where("LOWER(email) = LOWER(?) AND is_active = true", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Respuesta genérica — registrar para detección de abuse pero no filtrar al cliente
			log.Warn().Str("email", email).Msg("password reset requested for unknown email")
			return nil
		}
		return &responses.InternalResponse{Error: err, Message: "Error al consultar usuario", Handled: false}
	}

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		// 2. Invalidar tokens activos previos del usuario
		if err := tx.Exec(
			`UPDATE password_reset_tokens SET used_at = NOW() WHERE user_id = ? AND used_at IS NULL`,
			user.ID,
		).Error; err != nil {
			return fmt.Errorf("invalidar tokens previos: %w", err)
		}

		// 3. Generar token seguro (32 bytes → 64 hex chars)
		token, err := tools.GenerateSecureToken(32)
		if err != nil {
			return fmt.Errorf("generar token: %w", err)
		}

		id, err := tools.GenerateNanoid(tx)
		if err != nil {
			return fmt.Errorf("generar id: %w", err)
		}

		prt := database.PasswordResetToken{
			ID:        id,
			UserID:    user.ID,
			Token:     token,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		if err := tx.Create(&prt).Error; err != nil {
			return fmt.Errorf("crear reset token: %w", err)
		}

		// 4. Enviar email (no bloquear si falla — el token ya está creado)
		appURL := r.Config.AppURL
		if appURL == "" {
			appURL = "http://localhost:4200"
		}
		resetLink := fmt.Sprintf("%s/reset-password?token=%s", appURL, token)
		userName := user.FirstName
		if userName == "" {
			userName = user.Name
		}
		if emailSender := r.EmailSender; emailSender != nil {
			if err := emailSender.SendPasswordReset(user.Email, userName, resetLink); err != nil {
				log.Error().Err(err).Str("user_id", user.ID).Msg("email send failed — token still valid")
			}
		}

		return nil
	})

	if txErr != nil {
		return &responses.InternalResponse{Error: txErr, Message: "Error al procesar solicitud", Handled: false}
	}

	// 5. Audit log — fire-and-forget fuera del tx (AuditService.Log usa goroutine interna)
	if r.AuditService != nil {
		r.AuditService.Log(ctx, &user.ID, "password_reset_requested", "user", user.ID, nil, nil, "", "")
	}
	return nil
}

func (r *AuthenticationRepository) ResetPassword(ctx context.Context, token, newPassword string) *responses.InternalResponse {
	// 1. Validar el token
	var prt database.PasswordResetToken
	err := r.DB.Where("token = ? AND used_at IS NULL AND expires_at > NOW()", token).First(&prt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &responses.InternalResponse{
				Message:    "El enlace es inválido o expiró. Solicita uno nuevo.",
				Handled:    true,
				StatusCode: responses.StatusBadRequest,
			}
		}
		return &responses.InternalResponse{Error: err, Message: "Error al validar token", Handled: false}
	}

	// 2. Hashear la nueva contraseña usando Encrypt (mismo esquema que Login/ComparePasswords)
	hashed, err := tools.Encrypt(newPassword, r.JWTSecret)
	if err != nil {
		return &responses.InternalResponse{Error: err, Message: "Error al procesar contraseña", Handled: false}
	}

	txErr := r.DB.Transaction(func(tx *gorm.DB) error {
		// 3. Actualizar contraseña del usuario
		if err := tx.Exec(
			`UPDATE users SET password = ?, updated_at = NOW() WHERE id = ?`,
			hashed, prt.UserID,
		).Error; err != nil {
			return fmt.Errorf("actualizar contraseña: %w", err)
		}

		// 4. Marcar token como usado
		if err := tx.Exec(
			`UPDATE password_reset_tokens SET used_at = NOW() WHERE id = ?`,
			prt.ID,
		).Error; err != nil {
			return fmt.Errorf("marcar token usado: %w", err)
		}

		// 5. Invalidar TODAS las sesiones activas del usuario (mitigación account takeover)
		if err := tx.Exec(
			`UPDATE sessions SET is_active = false, updated_at = NOW() WHERE user_id = ? AND is_active = true`,
			prt.UserID,
		).Error; err != nil {
			return fmt.Errorf("invalidar sesiones: %w", err)
		}

		return nil
	})

	if txErr != nil {
		return &responses.InternalResponse{Error: txErr, Message: "Error al actualizar contraseña", Handled: false}
	}

	// 6. Audit log — fire-and-forget fuera del tx
	if r.AuditService != nil {
		r.AuditService.Log(ctx, &prt.UserID, "password_reset_completed", "user", prt.UserID, nil, nil, "", "")
	}
	return nil
}
