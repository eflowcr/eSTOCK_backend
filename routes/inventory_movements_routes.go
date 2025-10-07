package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterInventoryMovementsRoutes(router *gin.RouterGroup, db *gorm.DB) {
	inventoryMovementsRepository := &repositories.InventoryMovementsRepository{DB: db}
	inventoryMovementsService := services.NewInventoryMovementsService(inventoryMovementsRepository)

	inventoryMovementsController := controllers.NewInventoryMovementsController(*inventoryMovementsService)

	inventoryMovementsRoute := router.Group("/inventory_movements")
	{
		inventoryMovementsRoute.GET("/", inventoryMovementsController.GetAllInventoryMovements)
		inventoryMovementsRoute.GET("/:sku", inventoryMovementsController.GetMovementsBySku)
	}
}
