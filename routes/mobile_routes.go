package routes

import (
	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
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
	auditSvc *services.AuditService,
	notifSvc *services.NotificationsService,
) {
	// Build services (reuse existing wire helpers; one allocation per request lifecycle is fine).
	// W0.6: dev sprint-s2 made auditSvc/notifSvc required deps for picking + receiving.
	// Post-merge: NewPickingTask gained SOPickedQtyUpdater (SO3 cross-domain link) — pass via NewSalesOrders.
	soRepo, _ := wire.NewSalesOrders(db, config)
	_, pickingSvc := wire.NewPickingTask(db, auditSvc, notifSvc, soRepo)
	_, receivingSvc := wire.NewReceivingTasks(db, notifSvc)
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
					transfersSvc = services.NewStockTransfersServiceWithExecute(transferRepo, locRepo, db, config.TenantID)
				}
			}
			if transfersSvc == nil {
				transfersSvc = base
			}
		}
	}

	// Halo v1: read-only enrichment deps for the transfers list (location UUID →
	// code, assignee UUID → name). Web routes untouched.
	_, locationsSvc := wire.NewLocations(db, pool)
	_, usersSvc := wire.NewUsers(db, config, notifSvc)
	_, articlesSvc := wire.NewArticles(db, pool)

	mobileCtrl := controllers.NewMobileController(pickingSvc, receivingSvc, transfersSvc, inventorySvc, movementsSvc, alertsSvc, locationsSvc, usersSvc, articlesSvc, config)

	// Users admin (mobile-only, admin-gated). Reuses the SAME UserService the web
	// path uses so password hashing (tools.Encrypt) + validation are identical.
	// rolesRepo backs the form's role picker; it is nil in sqlserver/test mode and
	// the controller nil-guards it.
	usersAdminCtrl := controllers.NewMobileUsersController(usersSvc, rolesRepo, config)

	// Counts service & controller (mobile-only).
	_, countsSvc := wire.NewInventoryCounts(db, pool)
	countsCtrl := controllers.NewInventoryCountsController(*countsSvc, config.JWTSecret)
	// Halo v1: resolve detail-line SKU → product name (read-only, reuses the
	// already-wired articlesSvc). Nil-safe in test mode.
	countsCtrl.Articles = articlesSvc

	// S7.2 W0 — Idempotency-Key middleware for mobile write paths. Wraps the
	// 5 mutation endpoints that mobile can replay from its offline outbox.
	// Repo may be nil when db is nil (test mode) — middleware handles that.
	var idempotencyRepo ports.IdempotencyKeysRepository
	if db != nil {
		idempotencyRepo = &repositories.IdempotencyKeysRepository{DB: db}
	}

	mobile := router.Group("/mobile")
	mobile.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		readInventory := tools.RequirePermission(rolesRepo, "inventory", "read")
		updateInventory := tools.RequirePermission(rolesRepo, "inventory", "update")
		// Users admin is ADMIN-gated, not inventory-gated: only roles with the
		// "users" resource permission (or permissions.all → admin) may list/manage
		// users. Mirrors the web /api/users + /api/roles permission resources.
		readUsers := tools.RequirePermission(rolesRepo, "users", "read")
		updateUsers := tools.RequirePermission(rolesRepo, "users", "update")
		// Mount AFTER auth + permission so we never cache 401/403 responses
		// (those depend on the token, not the request body).
		dedupeMutation := tools.IdempotencyMiddleware(idempotencyRepo)

		// Health (no permission check — just JWT validation).
		mobile.GET("/health", mobileCtrl.Health)

		// Picking
		mobile.GET("/picking-tasks", readInventory, mobileCtrl.ListPickingTasks)
		mobile.GET("/picking-tasks/:id", readInventory, mobileCtrl.GetPickingTask)
		mobile.PATCH("/picking-tasks/:id/start", updateInventory, mobileCtrl.StartPickingTask)
		mobile.PATCH("/picking-tasks/:id/complete-line", updateInventory, dedupeMutation, mobileCtrl.CompletePickingLine)
		mobile.PATCH("/picking-tasks/:id/complete", updateInventory, mobileCtrl.CompletePickingTask)

		// Receiving
		mobile.GET("/receiving-tasks", readInventory, mobileCtrl.ListReceivingTasks)
		mobile.GET("/receiving-tasks/:id", readInventory, mobileCtrl.GetReceivingTask)
		mobile.PATCH("/receiving-tasks/:id/complete-line", updateInventory, dedupeMutation, mobileCtrl.CompleteReceivingLine)
		mobile.PATCH("/receiving-tasks/:id/complete", updateInventory, mobileCtrl.CompleteReceivingTask)

		// Stock Transfers
		mobile.GET("/stock-transfers", readInventory, mobileCtrl.ListStockTransfers)
		mobile.GET("/stock-transfers/:id", readInventory, mobileCtrl.GetStockTransfer)
		mobile.POST("/stock-transfers/:id/execute", updateInventory, dedupeMutation, mobileCtrl.ExecuteStockTransfer)

		// Inventory query
		mobile.GET("/inventory", readInventory, mobileCtrl.QueryInventory)
		mobile.GET("/inventory/sku/:sku/lots", readInventory, mobileCtrl.GetLotsBySKU)
		mobile.GET("/inventory/sku/:sku/movements", readInventory, mobileCtrl.GetMovementsBySKU)

		// Recent activity feed (Historial tab) — tenant-wide, newest first.
		mobile.GET("/movements", readInventory, mobileCtrl.GetRecentMovements)

		// Stock alerts (read-only)
		mobile.GET("/stock-alerts", readInventory, mobileCtrl.ListStockAlerts)

		// Users admin (mobile-only, admin-gated). Role list is read-gated (the form
		// needs it). Web /api/users + /api/roles routes are untouched.
		mobile.GET("/users", readUsers, usersAdminCtrl.ListUsers)
		mobile.POST("/users", updateUsers, usersAdminCtrl.CreateUser)
		mobile.PATCH("/users/:id", updateUsers, usersAdminCtrl.UpdateUser)
		mobile.GET("/roles", readUsers, usersAdminCtrl.ListRoles)

		// Counts (mobile-only module)
		counts := mobile.Group("/counts")
		{
			counts.GET("", readInventory, countsCtrl.List)
			counts.GET("/:id", readInventory, countsCtrl.GetDetail)
			counts.POST("", updateInventory, countsCtrl.Create)
			counts.PATCH("/:id/start", updateInventory, countsCtrl.Start)
			counts.POST("/:id/scan-line", updateInventory, dedupeMutation, countsCtrl.ScanLine)
			counts.POST("/:id/submit", updateInventory, dedupeMutation, countsCtrl.Submit)
			counts.PATCH("/:id/cancel", updateInventory, countsCtrl.Cancel)
		}
	}
}
