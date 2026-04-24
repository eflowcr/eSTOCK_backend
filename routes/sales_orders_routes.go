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

// RegisterSalesOrdersRoutes wires up /api/sales-orders endpoints.
func RegisterSalesOrdersRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	_, svc := wire.NewSalesOrders(db, config)
	ctrl := controllers.NewSalesOrdersController(svc, config.JWTSecret, config.TenantID)

	route := router.Group("/sales-orders")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read  := tools.RequirePermission(rolesRepo, "sales_orders", "read")
		write := tools.RequirePermission(rolesRepo, "sales_orders", "write")

		// SO1 — CRUD
		route.GET("", read, ctrl.List)
		route.GET("/:id", read, ctrl.GetByID)
		route.POST("", write, ctrl.Create)
		route.PATCH("/:id", write, ctrl.Update)
		route.DELETE("/:id", write, ctrl.SoftDelete)

		// SO2 — Lifecycle
		route.PATCH("/:id/submit", write, ctrl.Submit)
		route.PATCH("/:id/cancel", write, ctrl.Cancel)
	}
}
