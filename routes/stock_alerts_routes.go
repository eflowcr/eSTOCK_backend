package routes

import (
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/eflowcr/eSTOCK_backend/controllers"
	"github.com/eflowcr/eSTOCK_backend/ports"
	"github.com/eflowcr/eSTOCK_backend/repositories"
	"github.com/eflowcr/eSTOCK_backend/tools"
	"github.com/eflowcr/eSTOCK_backend/wire"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

var _ ports.StockAlertsRepository = (*repositories.StockAlertsRepository)(nil)

func RegisterStockAlertsRoutes(router *gin.RouterGroup, db *gorm.DB, config configuration.Config, redisClient *goredis.Client, rolesRepo ports.RolesRepository) {
	_, stockAlertsService := wire.NewStockAlerts(db, redisClient)
	// S3.5 W2-B: TenantID flows from configuration.Config into the controller and into
	// every service/repo call. Cron callers (admin_cron_controller, main goroutine) wire
	// tenantID separately by iterating the tenants table.
	stockAlertsController := controllers.NewStockAlertsController(*stockAlertsService, config.TenantID)

	// 5 requests per minute per IP on the analyze endpoint (expensive: full DB scan + transaction).
	analyzeRateLimiter := tools.NewIPRateLimiter(rate.Every(time.Minute/5), 5)

	route := router.Group("/stock-alerts")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "stock_alerts", "read")
		update := tools.RequirePermission(rolesRepo, "stock_alerts", "update")

		// S3.5.4 (B15 fix): root listing now responds (defaults to active alerts).
		// Previously GET /stock-alerts/ returned 404 because only /:resolved existed,
		// breaking probes / frontend search() helper / endpoint discovery. The /:resolved
		// variant remains for explicit filtering (true|false).
		route.GET("", read, stockAlertsController.GetAllStockAlerts)
		route.GET("/", read, stockAlertsController.GetAllStockAlerts)
		route.GET("/analyze", read, analyzeRateLimiter, stockAlertsController.Analyze)
		route.GET("/lot-expiration", read, stockAlertsController.LotExpiration)
		route.GET("/export", read, stockAlertsController.ExportAlertsToExcel)
		route.GET("/:resolved", read, stockAlertsController.GetAllStockAlerts)
		route.PATCH("/:id/resolve", update, stockAlertsController.ResolveAlert)
	}
}
