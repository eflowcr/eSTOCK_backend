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
		read           := tools.RequirePermission(rolesRepo, "backorders", "read")
		// fulfill creates a picking task — gate on picking_tasks:create (more accurate than backorders:create).
		fulfillPerm    := tools.RequirePermission(rolesRepo, "picking_tasks", "create")

		// BO2 — list + detail + fulfill
		route.GET("", read, ctrl.List)
		route.GET("/:id", read, ctrl.GetByID)
		route.POST("/:id/fulfill", fulfillPerm, ctrl.Fulfill)
	}
}
