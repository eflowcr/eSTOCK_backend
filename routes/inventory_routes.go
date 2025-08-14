package routes

import (
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterInventoryRoutes(router *gin.RouterGroup, db *gorm.DB) {
	inventoryRepository := &repositories.InventoryRepository{DB: db}
	inventoryService := services.NewInventoryService(inventoryRepository)

	inventoryController := controllers.NewInventoryController(*inventoryService)

	route := router.Group("/inventory")
	route.Use(tools.JWTAuthMiddleware())
	{
		route.GET("/", inventoryController.GetAllInventory)
		route.POST("/", inventoryController.CreateInventory)
		route.PUT("/:id", inventoryController.UpdateInventory)
		route.DELETE("/:id/:location", inventoryController.DeleteInventory)
		route.GET("/:sku/trend", inventoryController.Trend)
	}
}
