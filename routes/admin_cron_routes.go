package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterAdminCronRoutes registers POST /api/admin/cron/trigger.
// Requires JWT authentication and "cron":"trigger" permission (admin roles with {"all":true} qualify).
func RegisterAdminCronRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	ctrl := controllers.NewAdminCronController(db)

	route := router.Group("/admin/cron")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	route.Use(tools.RequirePermission(rolesRepo, "cron", "trigger"))
	{
		route.POST("/trigger", ctrl.Trigger)
	}
}
