package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ ports.InventoryMovementsRepository = (*repositories.InventoryMovementsRepository)(nil)

func RegisterInventoryMovementsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, inventoryMovementsService := wire.NewInventoryMovements(db)
	inventoryMovementsController := controllers.NewInventoryMovementsController(*inventoryMovementsService)

	// Legacy per-SKU endpoint (kept for backward compat)
	inventoryMovementsRoute := router.Group("/inventory_movements")
	{
		inventoryMovementsRoute.GET("/:sku", inventoryMovementsController.GetAllInventoryMovements)
	}

	// Global movements endpoint with optional query filters (used by Stock Ledger + article History tab)
	globalRoute := router.Group("/inventory-movements")
	globalRoute.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		globalRoute.GET("/", inventoryMovementsController.ListMovements)
	}
}
