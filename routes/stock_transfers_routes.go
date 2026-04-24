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
	"gorm.io/gorm"
)

var _ ports.StockTransfersRepository = (*repositories.StockTransfersRepositorySQLC)(nil)

func RegisterStockTransfersRoutes(router *gin.RouterGroup, db *gorm.DB, pool *pgxpool.Pool, config configuration.Config, rolesRepo ports.RolesRepository, auditSvc *services.AuditService) {
	if pool == nil {
		return
	}
	transferRepo, baseSvc := wire.NewStockTransfers(pool)
	if transferRepo == nil {
		return
	}
	var svc *services.StockTransfersService
	if db != nil {
		locationsRepo, _ := wire.NewLocations(db, pool)
		if locationsRepo != nil {
			svc = services.NewStockTransfersServiceWithExecute(transferRepo, locationsRepo, db, config.TenantID)
		}
	}
	if svc == nil {
		svc = baseSvc
	}
	ctrl := controllers.NewStockTransfersController(*svc, config.JWTSecret, auditSvc)

	route := router.Group("/stock-transfers")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		readInventory := tools.RequirePermission(rolesRepo, "inventory", "read")
		updateInventory := tools.RequirePermission(rolesRepo, "inventory", "update")

		route.GET("/", readInventory, ctrl.ListStockTransfers)
		route.GET("/:id", readInventory, ctrl.GetStockTransferByID)
		route.POST("/", updateInventory, ctrl.CreateStockTransfer)
		route.PUT("/:id", updateInventory, ctrl.UpdateStockTransfer)
		route.DELETE("/:id", updateInventory, ctrl.DeleteStockTransfer)
		route.POST("/:id/execute", updateInventory, ctrl.ExecuteStockTransfer)

		route.GET("/:id/lines", readInventory, ctrl.ListStockTransferLines)
		route.POST("/:id/lines", updateInventory, ctrl.CreateStockTransferLine)
		route.PUT("/:id/lines/:lineId", updateInventory, ctrl.UpdateStockTransferLine)
		route.DELETE("/:id/lines/:lineId", updateInventory, ctrl.DeleteStockTransferLine)
	}
}
