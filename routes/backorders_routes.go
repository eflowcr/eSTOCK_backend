package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterBackordersRoutes wires /api/backorders endpoints (BO2).
func RegisterBackordersRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	if db == nil {
		return
	}
	_, svc := wire.NewBackorders(db)
	ctrl := controllers.NewBackordersController(svc, config.TenantID)

	route := router.Group("/backorders")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read   := tools.RequirePermission(rolesRepo, "backorders", "read")
		create := tools.RequirePermission(rolesRepo, "backorders", "create")

		// BO2 — list + detail + fulfill
		route.GET("", read, ctrl.List)
		route.GET("/:id", read, ctrl.GetByID)
		route.POST("/:id/fulfill", create, ctrl.Fulfill)
	}
}
