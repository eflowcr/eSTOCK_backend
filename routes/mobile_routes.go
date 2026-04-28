package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/services"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterMobileRoutes mounts the mobile-facing API surface under /api/mobile/*.
//
// Design rule: handlers are thin adapters over existing services. The /api/* routes for the
// web frontend are not touched by this group (HARD requirement from the W0 brief).
//
// Permission model:
//   - All routes require a valid JWT (JWTAuthMiddleware).
//   - Read endpoints (list/get) require permission "inventory.read".
//   - Write endpoints (start/complete-line/complete/execute/scan-line/submit) require "inventory.update".
//   - When rolesRepo is nil (e.g. sqlserver mode), permission middleware is a no-op (matches existing pattern in /api/stock-transfers).
func RegisterMobileRoutes(
	router *gin.RouterGroup,
	db *gorm.DB,
	pool *pgxpool.Pool,
	config configuration.Config,
	rolesRepo ports.RolesRepository,
	redisClient *goredis.Client,
) {
	// Build services (reuse existing wire helpers; one allocation per request lifecycle is fine).
	_, pickingSvc := wire.NewPickingTask(db)
	_, receivingSvc := wire.NewReceivingTasks(db)
	_, inventorySvc := wire.NewInventory(db, pool)
	_, movementsSvc := wire.NewInventoryMovements(db)
	_, alertsSvc := wire.NewStockAlerts(db, redisClient)

	var transfersSvc *services.StockTransfersService
	if pool != nil {
		transferRepo, base := wire.NewStockTransfers(pool)
		if transferRepo != nil {
			if db != nil {
				locRepo, _ := wire.NewLocations(db, pool)
				if locRepo != nil {
					transfersSvc = services.NewStockTransfersServiceWithExecute(transferRepo, locRepo, db)
				}
			}
			if transfersSvc == nil {
				transfersSvc = base
			}
		}
	}

	mobileCtrl := controllers.NewMobileController(pickingSvc, receivingSvc, transfersSvc, inventorySvc, movementsSvc, alertsSvc, config)

	// Counts service & controller (mobile-only).
	_, countsSvc := wire.NewInventoryCounts(db)
	countsCtrl := controllers.NewInventoryCountsController(*countsSvc, config.JWTSecret)

	mobile := router.Group("/mobile")
	mobile.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		readInventory := tools.RequirePermission(rolesRepo, "inventory", "read")
		updateInventory := tools.RequirePermission(rolesRepo, "inventory", "update")

		// Health (no permission check — just JWT validation).
		mobile.GET("/health", mobileCtrl.Health)

		// Picking
		mobile.GET("/picking-tasks", readInventory, mobileCtrl.ListPickingTasks)
		mobile.GET("/picking-tasks/:id", readInventory, mobileCtrl.GetPickingTask)
		mobile.PATCH("/picking-tasks/:id/start", updateInventory, mobileCtrl.StartPickingTask)
		mobile.PATCH("/picking-tasks/:id/complete-line", updateInventory, mobileCtrl.CompletePickingLine)
		mobile.PATCH("/picking-tasks/:id/complete", updateInventory, mobileCtrl.CompletePickingTask)

		// Receiving
		mobile.GET("/receiving-tasks", readInventory, mobileCtrl.ListReceivingTasks)
		mobile.GET("/receiving-tasks/:id", readInventory, mobileCtrl.GetReceivingTask)
		mobile.PATCH("/receiving-tasks/:id/complete-line", updateInventory, mobileCtrl.CompleteReceivingLine)
		mobile.PATCH("/receiving-tasks/:id/complete", updateInventory, mobileCtrl.CompleteReceivingTask)

		// Stock Transfers
		mobile.GET("/stock-transfers", readInventory, mobileCtrl.ListStockTransfers)
		mobile.GET("/stock-transfers/:id", readInventory, mobileCtrl.GetStockTransfer)
		mobile.POST("/stock-transfers/:id/execute", updateInventory, mobileCtrl.ExecuteStockTransfer)

		// Inventory query
		mobile.GET("/inventory", readInventory, mobileCtrl.QueryInventory)
		mobile.GET("/inventory/sku/:sku/lots", readInventory, mobileCtrl.GetLotsBySKU)
		mobile.GET("/inventory/sku/:sku/movements", readInventory, mobileCtrl.GetMovementsBySKU)

		// Stock alerts (read-only)
		mobile.GET("/stock-alerts", readInventory, mobileCtrl.ListStockAlerts)

		// Counts (mobile-only module)
		counts := mobile.Group("/counts")
		{
			counts.GET("", readInventory, countsCtrl.List)
			counts.GET("/:id", readInventory, countsCtrl.GetDetail)
			counts.POST("", updateInventory, countsCtrl.Create)
			counts.PATCH("/:id/start", updateInventory, countsCtrl.Start)
			counts.POST("/:id/scan-line", updateInventory, countsCtrl.ScanLine)
			counts.POST("/:id/submit", updateInventory, countsCtrl.Submit)
			counts.PATCH("/:id/cancel", updateInventory, countsCtrl.Cancel)
		}
	}
}
