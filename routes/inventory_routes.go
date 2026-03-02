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

var _ ports.InventoryRepository = (*repositories.InventoryRepository)(nil)

func RegisterInventoryRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config) {
	_, inventoryService := wire.NewInventory(db)
	inventoryController := controllers.NewInventoryController(*inventoryService, config.JWTSecret)

	route := router.Group("/inventory")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
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
