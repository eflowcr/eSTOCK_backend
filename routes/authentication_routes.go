package routes

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

var _ ports.AuthenticationRepository = (*repositories.AuthenticationRepository)(nil)

// RegisterAuthenticationRoutes registers auth routes. If rolesRepo is non-nil,
// login response includes permissions for the user's role. If auditSvc is non-nil,
// password reset events are recorded in the audit log.
func RegisterAuthenticationRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository, auditSvc *services.AuditService) {
	var authenticationService = buildAuthService(db, config, rolesRepo, auditSvc)
	authenticationController := controllers.NewAuthenticationController(*authenticationService)

	route := router.Group("/auth")
	{
		route.POST("/login", authenticationController.Login)

		// Rate limit agresivo en forgot/reset para evitar abuse:
		// - /forgot-password: 5 requests por hora por IP (previene enumeración de usuarios)
		// - /reset-password:  10 intentos por hora por IP (previene brute force del token)
		route.POST("/forgot-password",
			tools.NewIPRateLimiter(rate.Every(12*time.Minute), 5),
			authenticationController.ForgotPassword)
		route.POST("/reset-password",
			tools.NewIPRateLimiter(rate.Every(6*time.Minute), 10),
			authenticationController.ResetPassword)
	}
}

func buildAuthService(db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository, auditSvc *services.AuditService) *services.AuthenticationService {
	if auditSvc != nil {
		_, svc := wire.NewAuthenticationWithAudit(db, config, rolesRepo, auditSvc)
		return svc
	}
	if rolesRepo != nil {
		_, svc := wire.NewAuthenticationWithRoles(db, config, rolesRepo)
		return svc
	}
	_, svc := wire.NewAuthentication(db, config)
	return svc
}
