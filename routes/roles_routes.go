package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
)

// RegisterRolesRoutes registers GET /api/roles, GET /api/roles/:id, PUT /api/roles/:id.
// Requires "roles" read for GET, "roles" update for PUT (admin only by default).
func RegisterRolesRoutes(router *gin.RouterGroup, config configuration.Config, rolesRepo ports.RolesRepository) {
	if rolesRepo == nil {
		return
	}
	ctrl := controllers.NewRolesController(rolesRepo)
	route := router.Group("/roles")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/", tools.RequirePermission(rolesRepo, "roles", "read"), ctrl.ListRoles)
		route.GET("/:id", tools.RequirePermission(rolesRepo, "roles", "read"), ctrl.GetRoleByID)
		route.PUT("/:id", tools.RequirePermission(rolesRepo, "roles", "update"), ctrl.UpdateRolePermissions)
	}
}
