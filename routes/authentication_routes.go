package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ ports.AuthenticationRepository = (*repositories.AuthenticationRepository)(nil)

// RegisterAuthenticationRoutes registers auth routes. If rolesRepo is non-nil,
// login response includes permissions for the user's role.
func RegisterAuthenticationRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	var authenticationService *services.AuthenticationService
	if rolesRepo != nil {
		_, authenticationService = wire.NewAuthenticationWithRoles(db, config, rolesRepo)
	} else {
		_, authenticationService = wire.NewAuthentication(db, config)
	}
	authenticationController := controllers.NewAuthenticationController(*authenticationService)

	route := router.Group("/auth")
	{
		route.POST("/login", authenticationController.Login)
	}
}
