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

// RegisterPurchaseOrdersRoutes wires purchase order CRUD + lifecycle endpoints.
// Pattern mirrors RegisterClientsRoutes (pool-backed, rolesRepo for per-verb RBAC).
func RegisterPurchaseOrdersRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, rolesRepo ports.RolesRepository) {
	if db == nil {
		return
	}
	_, svc := wire.NewPurchaseOrders(db)
	ctrl := controllers.NewPurchaseOrdersController(svc, config.TenantID)

	route := router.Group("/purchase-orders")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read   := tools.RequirePermission(rolesRepo, "purchase_orders", "read")
		create := tools.RequirePermission(rolesRepo, "purchase_orders", "create")
		update := tools.RequirePermission(rolesRepo, "purchase_orders", "update")
		delete := tools.RequirePermission(rolesRepo, "purchase_orders", "delete")

		route.GET("/", read, ctrl.List)
		route.GET("/:id", read, ctrl.GetByID)
		route.POST("/", create, ctrl.Create)
		route.PATCH("/:id", update, ctrl.Update)
		route.DELETE("/:id", delete, ctrl.Delete)
		// PO2 lifecycle
		route.PATCH("/:id/submit", update, ctrl.Submit)
		route.PATCH("/:id/cancel", update, ctrl.Cancel)
	}
}
