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
	stockAlertsController := controllers.NewStockAlertsController(*stockAlertsService)

	// 5 requests per minute per IP on the analyze endpoint (expensive: full DB scan + transaction).
	analyzeRateLimiter := tools.NewIPRateLimiter(rate.Every(time.Minute/5), 5)

	route := router.Group("/stock-alerts")
	route.Use(tools.JWTAuthMiddleware(config.JWTSecret))
	{
		read := tools.RequirePermission(rolesRepo, "stock_alerts", "read")
		update := tools.RequirePermission(rolesRepo, "stock_alerts", "update")

		route.GET("/:resolved", read, stockAlertsController.GetAllStockAlerts)
		route.GET("/analyze", read, analyzeRateLimiter, stockAlertsController.Analyze)
		route.GET("/lot-expiration", read, stockAlertsController.LotExpiration)
		route.PATCH("/:id/resolve", update, stockAlertsController.ResolveAlert)
		route.GET("/export", read, stockAlertsController.ExportAlertsToExcel)
	}
}
