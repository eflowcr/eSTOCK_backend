package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/models/database"
	"github.com/eflowcr/eSTOCK_backend/models/requests"
	"github.com/eflowcr/eSTOCK_backend/models/responses"
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

	// S3.5 W3 — embed tenant_id into JWT. users table doesn't carry tenant_id yet
	// (single-tenant pilot), so the source is Config.TenantID. When the users table gains
	// tenant_id (planned post-S3.5), swap to user.TenantID without touching this signature.
	token, err := tools.GenerateToken(a.JWTSecret, user.ID, user.Name, user.Email, user.RoleID, a.Config.TenantID)
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
