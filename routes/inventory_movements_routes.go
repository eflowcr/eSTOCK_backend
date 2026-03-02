package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var _ ports.InventoryMovementsRepository = (*repositories.InventoryMovementsRepository)(nil)

func RegisterInventoryMovementsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, inventoryMovementsService := wire.NewInventoryMovements(db)
	inventoryMovementsController := controllers.NewInventoryMovementsController(*inventoryMovementsService)

	inventoryMovementsRoute := router.Group("/inventory_movements")
	{
		inventoryMovementsRoute.GET("/:sku", inventoryMovementsController.GetAllInventoryMovements)
	}
}
