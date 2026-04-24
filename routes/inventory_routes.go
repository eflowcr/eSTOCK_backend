package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

var _ ports.InventoryRepository = (*repositories.InventoryRepository)(nil)

func RegisterInventoryRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository) {
	// S3.5 W2-A: pass config so InventoryRepository stamps tenant_id on inventory_lots inserts.
	_, inventoryService := wire.NewInventoryWithConfig(db, pool, config)
	inventoryController := controllers.NewInventoryController(*inventoryService, config.JWTSecret)

	route := router.Group("/inventory")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		route.GET("/import/template", inventoryController.DownloadImportTemplate)
		route.POST("/import/validate", inventoryController.ValidateImportRows)
		route.POST("/import/json", inventoryController.ImportInventoryFromJSON)
		route.POST("/import", inventoryController.ImportInventoryFromExcel)
		route.GET("/export", inventoryController.ExportInventoryToExcel)

		route.GET("/", inventoryController.GetAllInventory)
		route.GET("/valuation", tools.RequirePermission(rolesRepo, "inventory", "read"), inventoryController.GetInventoryValuation)
		route.GET("/pick-suggestions/:sku", inventoryController.GetPickSuggestions)
		route.GET("/sku/:sku/location/:location", inventoryController.GetInventoryBySkuAndLocation)
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
