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
		route.POST("/import", inventoryController.ImportInventoryFromExcel)
		route.GET("/export", inventoryController.ExportInventoryToExcel)

		route.GET("/", inventoryController.GetAllInventory)
		route.POST("/", inventoryController.CreateInventory)

		id := route.Group("/id/:id")
		{
			id.PATCH("", inventoryController.UpdateInventory)
			id.DELETE("/:location", inventoryController.DeleteInventory)

			id.GET("/lots", inventoryController.GetInventoryLots)
			id.POST("/lots", inventoryController.CreateInventoryLot)
			id.DELETE("/lots/:lotId", inventoryController.DeleteInventoryLot)

			id.GET("/serials", inventoryController.GetInventorySerials)
			id.POST("/serials", inventoryController.CreateInventorySerial)
			id.DELETE("/serials", inventoryController.DeleteInventorySerial)
		}

		route.GET("/sku/:sku/trend", inventoryController.Trend)
	}
}
